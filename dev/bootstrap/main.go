package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

// bootstrap is a one-shot project renaming helper.
// Usage examples:
//
//	go run ./dev/bootstrap -module github.com/spacelink/awesome_app -app awesome_app
//	go run ./dev/bootstrap -app awesome_app
//	go run ./dev/bootstrap -module github.com/spacelink/awesome_app -dry-run
func main() {
	newModule := flag.String("module", "", "new Go module path (e.g., github.com/yourorg/yourapp)")
	newApp := flag.String("app", "", "new runtime APP name (used for logging, sockets, service name)")
	flagDryRun := flag.Bool("dry-run", false, "show planned changes without writing")
	flag.Parse()

	cwd, _ := os.Getwd()
	gomodPath := filepath.Join(cwd, "go.mod")
	gomodBytes, err := os.ReadFile(gomodPath)
	check(err, "read go.mod")

	mf, err := modfile.Parse("go.mod", gomodBytes, nil)
	check(err, "parse go.mod")
	if mf.Module == nil || mf.Module.Mod.Path == "" {
		fatal(errors.New("module path missing in go.mod"), "get current module path")
	}
	oldModule := mf.Module.Mod.Path

	// Plan summary for user visibility
	var plan []string
	if *newModule != "" && *newModule != oldModule {
		plan = append(plan, fmt.Sprintf("Change module: %s -> %s", oldModule, *newModule))
	} else {
		plan = append(plan, fmt.Sprintf("Module unchanged: %s", oldModule))
	}
	if *newApp != "" {
		plan = append(plan, fmt.Sprintf("Set APPNAME to: %s", *newApp))
	}
	fmt.Println("Bootstrap plan:")
	for _, p := range plan {
		fmt.Println(" -", p)
	}
	if *flagDryRun {
		fmt.Println("Dry run requested; no files will be modified.")
		return
	}

	// 1) Update go.mod module path
	if *newModule != "" && *newModule != oldModule {
		if err := mf.AddModuleStmt(*newModule); err != nil {
			fatal(err, "set new module path")
		}
		out, err := mf.Format()
		check(err, "format updated go.mod")
		err = os.WriteFile(gomodPath, out, 0o644)
		check(err, "write updated go.mod")
		fmt.Println("Updated go.mod module path")
	}

	// 2) Rewrite imports across .go files
	if *newModule != "" && *newModule != oldModule {
		countFiles, countImports, err := rewriteImports(cwd, oldModule, *newModule)
		check(err, "rewrite imports")
		fmt.Printf("Rewrote imports in %d files (%d import specs)\n", countFiles, countImports)
	}

	// 3) Update APPNAME in cmd/app/consts/consts.go
	if *newApp != "" {
		constsPath := filepath.Join(cwd, "cmd", "app", "consts", "consts.go")
		if _, err := os.Stat(constsPath); err == nil {
			updated, err := replaceAppName(constsPath, *newApp)
			check(err, "update APPNAME in consts.go")
			if updated {
				fmt.Println("Updated APPNAME in cmd/app/consts/consts.go")
			} else {
				fmt.Println("APPNAME not changed (no const APPNAME found?)")
			}
		} else if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Warning: consts.go not found; skipping APPNAME update")
		} else {
			check(err, "stat consts.go")
		}
	}

	// 4) Update .teamcity/settings.kts for app.name and ldflags module path and description
	teamCityPath := filepath.Join(cwd, ".teamcity", "settings.kts")
	if b, err := os.ReadFile(teamCityPath); err == nil {
		orig := string(b)
		repl := orig
		if *newApp != "" {
			// param("app.name", "...")
			repl = regexp.MustCompile(`param\("app.name",\s*"[^"]*"\)`).ReplaceAllString(repl, fmt.Sprintf(`param("app.name", "%s")`, *newApp))
			// description line like: description = "CI for <app>"
			repl = regexp.MustCompile(`description\s*=\s*"CI for [^"]*"`).ReplaceAllString(repl, fmt.Sprintf(`description = "CI for %s"`, pick(*newApp, oldModule)))
		}
		if *newModule != "" && *newModule != oldModule {
			// Replace occurrences of old module path in ldflags lines
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Version", *newModule+"/cmd/app/consts.Version")
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Commit", *newModule+"/cmd/app/consts.Commit")
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.BuildDate", *newModule+"/cmd/app/consts.BuildDate")
		}
		if repl != orig {
			err = os.WriteFile(teamCityPath, []byte(repl), 0o644)
			check(err, "write updated .teamcity/settings.kts")
			fmt.Println("Updated .teamcity/settings.kts")
		}
	}

	// 5) Update README.md with new app/module info
	updateReadme(cwd, oldModule, *newModule, *newApp)

	// 6) Best-effort: run `go mod tidy` to settle dependencies after rewrite
	if err := runGoModTidy(cwd); err != nil {
		fmt.Println("Note: go mod tidy failed:", err)
	}

	fmt.Println("Bootstrap completed successfully.")
}

func check(err error, context string) {
	if err != nil {
		fatal(err, context)
	}
}

func fatal(err error, context string) {
	fmt.Fprintf(os.Stderr, "bootstrap error (%s): %v\n", context, err)
	os.Exit(1)
}

func rewriteImports(root, oldModule, newModule string) (filesChanged int, importsChanged int, err error) {
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		// Skip vendor and .git and dev/bootstrap itself
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" || base == ".idea" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Parse file
		file, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return perr
		}
		changed := false
		for _, imp := range file.Imports {
			val := strings.Trim(imp.Path.Value, "\"")
			if strings.HasPrefix(val, oldModule) {
				newVal := newModule + strings.TrimPrefix(val, oldModule)
				if newVal != val {
					imp.Path.Value = fmt.Sprintf("\"%s\"", newVal)
					changed = true
					importsChanged++
				}
			}
		}
		if changed {
			var buf bytes.Buffer
			cfg := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
			if err := cfg.Fprint(&buf, fset, file); err != nil {
				return err
			}
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				return err
			}
			filesChanged++
		}
		return nil
	})
	return
}

func replaceAppName(constsPath, newApp string) (bool, error) {
	b, err := os.ReadFile(constsPath)
	if err != nil {
		return false, err
	}
	orig := string(b)
	re := regexp.MustCompile(`const\s+APPNAME\s*=\s*"[^"]*"`)
	repl := re.ReplaceAllString(orig, fmt.Sprintf("const APPNAME = \"%s\"", newApp))
	if repl == orig {
		return false, nil
	}
	return true, os.WriteFile(constsPath, []byte(repl), 0o644)
}

func runGoModTidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pick(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func updateReadme(cwd, oldModule, newModule, newApp string) {
	readmePath := filepath.Join(cwd, "README.md")
	b, err := os.ReadFile(readmePath)
	if err != nil {
		return // README might not exist; best-effort
	}
	orig := string(b)
	repl := orig

	oldAppName := filepath.Base(oldModule)
	if newApp != "" && oldAppName != "" {
		// Replace occurrences of the old app name with the new app name
		repl = strings.ReplaceAll(repl, oldAppName, newApp)
		// Also try a title pattern replacement if present: '# <name> —'
		reTitle := regexp.MustCompile(`(?m)^#\s*[^\n]*`)
		repl = reTitle.ReplaceAllStringFunc(repl, func(line string) string {
			// If the line already contains newApp, keep it; otherwise, replace oldAppName with newApp
			if strings.Contains(line, newApp) {
				return line
			}
			return strings.ReplaceAll(line, oldAppName, newApp)
		})
	}

	if newModule != "" && newModule != oldModule {
		// Update ldflags import paths shown in README examples
		repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Version", newModule+"/cmd/app/consts.Version")
		repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Commit", newModule+"/cmd/app/consts.Commit")
		repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.BuildDate", newModule+"/cmd/app/consts.BuildDate")
	}

	if repl != orig {
		_ = os.WriteFile(readmePath, []byte(repl), 0o644)
		fmt.Println("Updated README.md")
	}
}
