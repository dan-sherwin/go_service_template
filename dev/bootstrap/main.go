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
//	go run ./dev/bootstrap foo_bar
//	go run ./dev/bootstrap foo_bar -dry-run
func main() {
	flagDryRun := flag.Bool("dry-run", false, "show planned changes without writing")
	flag.Parse()

	// Expect exactly one positional argument: the app name (also used as module path)
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./dev/bootstrap <app_name> [-dry-run]")
		os.Exit(2)
	}
	appName := args[0]
	newModule := appName
	newApp := appName

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
	if newModule != "" && newModule != oldModule {
		plan = append(plan, fmt.Sprintf("Change module: %s -> %s", oldModule, newModule))
	} else {
		plan = append(plan, fmt.Sprintf("Module unchanged: %s", oldModule))
	}
	if newApp != "" {
		plan = append(plan, fmt.Sprintf("Set APPNAME to: %s", newApp))
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
	if newModule != "" && newModule != oldModule {
		if err := mf.AddModuleStmt(newModule); err != nil {
			fatal(err, "set new module path")
		}
		out, err := mf.Format()
		check(err, "format updated go.mod")
		err = os.WriteFile(gomodPath, out, 0o644)
		check(err, "write updated go.mod")
		fmt.Println("Updated go.mod module path")
	}

	// 2) Rewrite imports across .go files
	if newModule != "" && newModule != oldModule {
		countFiles, countImports, err := rewriteImports(cwd, oldModule, newModule)
		check(err, "rewrite imports")
		fmt.Printf("Rewrote imports in %d files (%d import specs)\n", countFiles, countImports)
	}

	// 3) Update APPNAME in cmd/app/consts/consts.go
	if newApp != "" {
		constsPath := filepath.Join(cwd, "cmd", "app", "consts", "consts.go")
		if _, err := os.Stat(constsPath); err == nil {
			updated, err := replaceAppName(constsPath, newApp)
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
		if newApp != "" {
			// param("app.name", "...")
			repl = regexp.MustCompile(`param\("app.name",\s*"[^"]*"\)`).ReplaceAllString(repl, fmt.Sprintf(`param("app.name", "%s")`, newApp))
			// description line like: description = "CI for <app>"
			repl = regexp.MustCompile(`description\s*=\s*"CI for [^"]*"`).ReplaceAllString(repl, fmt.Sprintf(`description = "CI for %s"`, pick(newApp, oldModule)))
		}
		if newModule != "" && newModule != oldModule {
			// Replace occurrences of old module path in ldflags lines
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Version", newModule+"/cmd/app/consts.Version")
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.Commit", newModule+"/cmd/app/consts.Commit")
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts.BuildDate", newModule+"/cmd/app/consts.BuildDate")
			repl = strings.ReplaceAll(repl, oldModule+"/cmd/app/consts", newModule+"/cmd/app/consts")
		}
		if repl != orig {
			err = os.WriteFile(teamCityPath, []byte(repl), 0o644)
			check(err, "write updated .teamcity/settings.kts")
			fmt.Println("Updated .teamcity/settings.kts")
		}
	}

	// 5) Update README.md with new app/module info
	updateReadme(cwd, oldModule, newModule, newApp)

	// 6) Update GoLand run configurations under dev/runConfigurations
	updateRunConfigurations(cwd, oldModule, newModule, newApp)

	// 7) Best-effort: run `go mod tidy` to settle dependencies after rewrite
	if err := runGoModTidy(cwd); err != nil {
		fmt.Println("Note: go mod tidy failed:", err)
	}

	// 8) Update .gormdb2struct.toml OutPackagePath to the correct module path
	gormConf := filepath.Join(cwd, ".gormdb2struct.toml")
	if b, err := os.ReadFile(gormConf); err == nil {
		orig := string(b)
		// Determine effective module path
		effectiveModule := oldModule
		if newModule != "" {
			effectiveModule = newModule
		}
		correct := fmt.Sprintf("OutPackagePath = \"%s/app/db\"", effectiveModule)
		re := regexp.MustCompile(`(?m)^OutPackagePath\s*=\s*"[^"]*"`)
		repl := re.ReplaceAllString(orig, correct)
		if repl != orig {
			if err := os.WriteFile(gormConf, []byte(repl), 0o644); err == nil {
				fmt.Println("Updated .gormdb2struct.toml OutPackagePath")
			}
		}
	}

	// 9) Update ci-local.sh and .golangci.yml if they exist
	updateCIConfig(cwd, oldModule, newModule)

	fmt.Println("Bootstrap completed successfully.")
}

func updateCIConfig(cwd, oldModule, newModule string) {
	if newModule == "" || newModule == oldModule {
		return
	}

	// Update .golangci.yml
	golangciPath := filepath.Join(cwd, ".golangci.yml")
	if b, err := os.ReadFile(golangciPath); err == nil {
		orig := string(b)
		// Replace old module path if it appears in path exclusions or rules
		repl := strings.ReplaceAll(orig, oldModule, newModule)
		if repl != orig {
			if err := os.WriteFile(golangciPath, []byte(repl), 0o644); err == nil {
				fmt.Println("Updated .golangci.yml")
			}
		}
	}

	// Update dev/ci-local.sh
	ciLocalPath := filepath.Join(cwd, "dev", "ci-local.sh")
	if b, err := os.ReadFile(ciLocalPath); err == nil {
		orig := string(b)
		repl := strings.ReplaceAll(orig, oldModule, newModule)
		if repl != orig {
			if err := os.WriteFile(ciLocalPath, []byte(repl), 0o644); err == nil {
				fmt.Println("Updated dev/ci-local.sh")
			}
		}
	}
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
	if _, err := os.Stat(readmePath); err != nil {
		// README might not exist; best-effort
		return
	}

	// Determine effective module path and app name
	effectiveModule := oldModule
	if newModule != "" {
		effectiveModule = newModule
	}
	appName := newApp
	if appName == "" {
		// Fall back to module basename
		appName = filepath.Base(effectiveModule)
		if appName == "" {
			appName = "app"
		}
	}

	// Construct a brand new README.md that is specific to the app, with no template mentions
	ldflagsVersion := effectiveModule + "/cmd/app/consts.Version"
	ldflagsCommit := effectiveModule + "/cmd/app/consts.Commit"
	ldflagsBuildDate := effectiveModule + "/cmd/app/consts.BuildDate"

	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(appName)
	b.WriteString(" — Go service with CLI, RPC, HTTP, logging, and systemd integration\n\n")

	b.WriteString("This repository contains the ")
	b.WriteString(appName)
	b.WriteString(" service. It provides:\n")
	b.WriteString("- A CLI using Kong\n")
	b.WriteString("- A Unix RPC server (net/rpc over a Unix domain socket)\n")
	b.WriteString("- An HTTP server you can extend with endpoints\n")
	b.WriteString("- Structured logging with slog (Journald on Linux; Text to stdout on macOS)\n")
	b.WriteString("- System data sampling (goroutine count, memory, CPU%)\n")
	b.WriteString("- Systemd integration using takama/daemon\n")
	b.WriteString("- Settings persistence\n")
	b.WriteString("- TeamCity CI setup\n\n")

	b.WriteString("## Quick start\n")
	b.WriteString("Build and run locally:\n\n")
	b.WriteString("```\n")
	b.WriteString("go build -o ./dist/")
	b.WriteString(appName)
	b.WriteString(" ./cmd\n")
	b.WriteString("./dist/")
	b.WriteString(appName)
	b.WriteString(" run\n")
	b.WriteString("```\n\n")

	b.WriteString("Linux production build example:\n\n")
	b.WriteString("```\n")
	b.WriteString("GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \\\n")
	b.WriteString("  go build -ldflags \"-X '")
	b.WriteString(ldflagsVersion)
	b.WriteString("=0.1.0' -X '")
	b.WriteString(ldflagsCommit)
	b.WriteString("=dev' -X '")
	b.WriteString(ldflagsBuildDate)
	b.WriteString("=$(date -u +%Y-%m-%dT%H:%M:%SZ)'\" -o ./dist/")
	b.WriteString(appName)
	b.WriteString(" ./cmd\n")
	b.WriteString("```\n\n")

	b.WriteString("The binary exposes a CLI with commands registered under cmd/app/commands. See internal/foo for examples of adding a command and a setting.\n\n")

	b.WriteString("## Logging\n")
	b.WriteString("- Linux: slog handler writes to journald (see cmd/app/logger_linux.go)\n")
	b.WriteString("- macOS: slog uses TextHandler to stdout (see cmd/app/logger_darwin.go)\n")
	b.WriteString("- Standard log keys: app, version, pid, user, component, error\n")
	b.WriteString("- Set log level via CLI or environment\n\n")

	b.WriteString("## RPC\n")
	b.WriteString("- Unix domain socket: ")
	b.WriteString(pick(os.Getenv("XDG_RUNTIME_DIR"), "/tmp"))
	b.WriteString("/")
	b.WriteString(appName)
	b.WriteString("/")
	b.WriteString(appName)
	b.WriteString("-rpc.sock (0660 perms)\n")
	b.WriteString("- Start server as part of daemon run path; client helpers dial per call and close\n\n")

	b.WriteString("## HTTP\n")
	b.WriteString("- HTTP server starts after daemon setup; add your endpoints as needed\n")
	b.WriteString("- Health checks: /healthz and /ready endpoints are registered by default\n\n")

	b.WriteString("## Systemd integration\n")
	b.WriteString("- Install/remove/start/stop/status via CLI under the Systemd command group\n")
	b.WriteString("- Uses takama/daemon to register the service\n\n")

	b.WriteString("## Versioning\n")
	b.WriteString("- Build info is injected via -ldflags into cmd/app/consts (Version, Commit, BuildDate)\n\n")

	b.WriteString("## CI\n")
	b.WriteString("- .teamcity/settings.kts contains a build that runs go mod tidy, go vet, go test -race, and a Linux/amd64 build with ldflags\n")
	b.WriteString("- Optional Deploy configuration can use rsync/SSH and systemctl restart; set parameters as needed (dest_user, dest_host, dest_path)\n\n")

	content := b.String()
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err == nil {
		fmt.Println("Updated README.md")
	}
}

// updateRunConfigurations updates IntelliJ GoLand runConfigurations to reflect the new module/app names.
func updateRunConfigurations(cwd, oldModule, newModule, newApp string) {
	dir := filepath.Join(cwd, "dev", "runConfigurations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory might not exist; nothing to do
		return
	}

	// Determine effective values
	effectiveModule := oldModule
	if newModule != "" {
		effectiveModule = newModule
	}
	ideaModuleName := filepath.Base(effectiveModule)
	appName := newApp
	if appName == "" {
		appName = ideaModuleName
		if appName == "" {
			appName = "app"
		}
	}

	reOldPkg := regexp.MustCompile(regexp.QuoteMeta(oldModule))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".run.xml") {
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		orig := string(b)
		repl := orig

		// 1) Update <module name="..." />
		repl = regexp.MustCompile(`<module\s+name="[^"]*"\s*/>`).ReplaceAllString(repl, fmt.Sprintf(`<module name="%s" />`, ideaModuleName))

		// 2) Update package path if it matches old module path
		if newModule != "" && newModule != oldModule {
			// Only replace exact attribute values that contain the old module path
			repl = regexp.MustCompile(`<package\s+value="`+regexp.QuoteMeta(oldModule)+`[^"]*"\s*/>`).ReplaceAllStringFunc(repl, func(s string) string {
				return strings.ReplaceAll(s, oldModule, newModule)
			})
			// Also replace any free-text occurrences just in case
			repl = reOldPkg.ReplaceAllString(repl, newModule)
		}

		// 3) Update output binary name for main app run config: build/service_template -> build/<appName>
		repl = strings.ReplaceAll(repl, "build/service_template", "build/"+appName)

		if repl != orig {
			_ = os.WriteFile(path, []byte(repl), 0o644)
			fmt.Println("Updated run configuration:", filepath.Join("dev/runConfigurations", name))
		}
	}
}
