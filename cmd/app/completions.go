package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"github.com/riywo/loginshell"
)

type (
	CompletionsCommandDef struct {
		Install   InstallCompletionsCommand   `cmd:"" help:"Persistently install shell completions for the detected shell"`
		Uninstall UninstallCompletionsCommand `cmd:"" help:"Remove previously installed shell completions for the detected shell"`
	}

	InstallCompletionsCommand struct {
		Shell   string `name:"shell" help:"Override the detected login shell (bash|zsh|fish)"`
		BinPath string `name:"bin-path" help:"Absolute path to the binary to register for shell completions"`
	}

	UninstallCompletionsCommand struct {
		Shell string `name:"shell" help:"Override the detected login shell (bash|zsh|fish)"`
	}
)

const (
	completionsMarkerStart = "# >>> %s completions >>>"
	completionsMarkerEnd   = "# <<< %s completions <<<"
)

func (c *InstallCompletionsCommand) Run() error {
	result, err := installCompletions(c.Shell, c.BinPath)
	if err != nil {
		return err
	}

	fmt.Printf("Installed %s completions for %s in %s\n", result.shell, consts.APPNAME, result.targetPath)
	fmt.Printf("Restart your shell or run: %s\n", result.reloadHint)
	return nil
}

func (c *UninstallCompletionsCommand) Run() error {
	result, err := uninstallCompletions(c.Shell)
	if err != nil {
		return err
	}

	if !result.changed {
		fmt.Printf("No installed %s completions were found for %s in %s\n", result.shell, consts.APPNAME, result.targetPath)
		return nil
	}

	fmt.Printf("Removed %s completions for %s from %s\n", result.shell, consts.APPNAME, result.targetPath)
	fmt.Printf("Restart your shell or run: %s\n", result.reloadHint)
	return nil
}

type completionsResult struct {
	shell      string
	targetPath string
	reloadHint string
	changed    bool
}

func installCompletions(shellOverride, binPathOverride string) (*completionsResult, error) {
	shellName, err := detectCompletionShell(shellOverride)
	if err != nil {
		return nil, err
	}

	binPath, err := resolveCompletionBinaryPath(binPathOverride)
	if err != nil {
		return nil, err
	}

	targetPath, reloadHint, err := completionTarget(shellName)
	if err != nil {
		return nil, err
	}

	fragment, err := completionFragment(shellName, binPath)
	if err != nil {
		return nil, err
	}

	if shellName == "fish" {
		if err := writeCompletionsFile(targetPath, fragment); err != nil {
			return nil, err
		}
	} else {
		block := managedCompletionsBlock(fragment)
		if err := upsertManagedCompletionsBlock(targetPath, block); err != nil {
			return nil, err
		}
	}

	return &completionsResult{
		shell:      shellName,
		targetPath: targetPath,
		reloadHint: reloadHint,
		changed:    true,
	}, nil
}

func uninstallCompletions(shellOverride string) (*completionsResult, error) {
	shellName, err := detectCompletionShell(shellOverride)
	if err != nil {
		return nil, err
	}

	targetPath, reloadHint, err := completionTarget(shellName)
	if err != nil {
		return nil, err
	}

	var changed bool
	if shellName == "fish" {
		err = os.Remove(targetPath)
		if err == nil {
			changed = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove completions file %s: %w", targetPath, err)
		}
	} else {
		changed, err = removeManagedCompletionsBlock(targetPath)
		if err != nil {
			return nil, err
		}
	}

	return &completionsResult{
		shell:      shellName,
		targetPath: targetPath,
		reloadHint: reloadHint,
		changed:    changed,
	}, nil
}

func detectCompletionShell(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if shell, err := loginshell.Shell(); err == nil && shell != "" {
		return filepath.Base(shell), nil
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell), nil
	}
	return "", fmt.Errorf("could not determine login shell; use --shell")
}

func resolveCompletionBinaryPath(override string) (string, error) {
	if override != "" {
		path, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve --bin-path: %w", err)
		}
		return path, nil
	}

	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	if strings.Contains(path, string(filepath.Separator)+"go-build"+string(filepath.Separator)) {
		return "", fmt.Errorf("current executable path %q is temporary; rerun with --bin-path pointing at the real binary", path)
	}
	return path, nil
}

func completionTarget(shellName string) (targetPath, reloadHint string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("determine home directory: %w", err)
	}

	switch shellName {
	case "bash":
		rcFile := ".bashrc"
		if runtime.GOOS == "darwin" {
			rcFile = ".bash_profile"
		}
		targetPath = filepath.Join(homeDir, rcFile)
	case "zsh":
		targetPath = filepath.Join(homeDir, ".zshrc")
	case "fish":
		targetPath = filepath.Join(homeDir, ".config", "fish", "conf.d", consts.APPNAME+"_completions.fish")
	default:
		return "", "", fmt.Errorf("unsupported shell %q", shellName)
	}

	return targetPath, "source " + shellQuote(targetPath), nil
}

func completionFragment(shellName, binPath string) (string, error) {
	quotedBin := shellQuote(binPath)

	switch shellName {
	case "bash":
		return fmt.Sprintf("complete -C %s %s\n", quotedBin, consts.APPNAME), nil
	case "zsh":
		return fmt.Sprintf("autoload -U +X bashcompinit && bashcompinit\ncomplete -C %s %s\n", quotedBin, consts.APPNAME), nil
	case "fish":
		return fmt.Sprintf("function __complete_%s\n    set -lx COMP_LINE (commandline -cp)\n    test -z (commandline -ct)\n    and set COMP_LINE \"$COMP_LINE \"\n    %s\nend\ncomplete -f -c %s -a \"(__complete_%s)\"\n", consts.APPNAME, quotedBin, consts.APPNAME, consts.APPNAME), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func managedCompletionsBlock(fragment string) string {
	return fmt.Sprintf(
		"%s\n%s%s\n",
		fmt.Sprintf(completionsMarkerStart, consts.APPNAME),
		strings.TrimRight(fragment, "\n"),
		"\n"+fmt.Sprintf(completionsMarkerEnd, consts.APPNAME),
	)
}

func upsertManagedCompletionsBlock(targetPath, block string) error {
	existing, err := readOptionalTextFile(targetPath)
	if err != nil {
		return err
	}
	updated := upsertManagedBlock(existing, block)
	return writeCompletionsFile(targetPath, updated)
}

func removeManagedCompletionsBlock(targetPath string) (bool, error) {
	existing, err := readOptionalTextFile(targetPath)
	if err != nil {
		return false, err
	}
	updated, removed := removeManagedBlock(existing)
	if !removed {
		return false, nil
	}
	return true, writeCompletionsFile(targetPath, updated)
}

func readOptionalTextFile(targetPath string) (string, error) {
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read completions file %s: %w", targetPath, err)
	}
	return string(data), nil
}

func writeCompletionsFile(targetPath, contents string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create completions directory for %s: %w", targetPath, err)
	}
	if err := os.WriteFile(targetPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write completions file %s: %w", targetPath, err)
	}
	return nil
}

func upsertManagedBlock(existing, block string) string {
	updated, removed := removeManagedBlock(existing)
	if removed {
		existing = updated
	}

	if strings.TrimSpace(existing) == "" {
		return block
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block
}

func removeManagedBlock(existing string) (string, bool) {
	startMarker := fmt.Sprintf(completionsMarkerStart, consts.APPNAME)
	endMarker := fmt.Sprintf(completionsMarkerEnd, consts.APPNAME)

	startIdx := strings.Index(existing, startMarker)
	if startIdx < 0 {
		return existing, false
	}
	relEndIdx := strings.Index(existing[startIdx:], endMarker)
	if relEndIdx < 0 {
		return existing, false
	}

	endIdx := startIdx + relEndIdx + len(endMarker)
	for endIdx < len(existing) && (existing[endIdx] == '\n' || existing[endIdx] == '\r') {
		endIdx++
	}

	prefix := strings.TrimRight(existing[:startIdx], "\n")
	suffix := strings.TrimLeft(existing[endIdx:], "\n")

	switch {
	case prefix == "" && suffix == "":
		return "", true
	case prefix == "":
		return suffix, true
	case suffix == "":
		return prefix + "\n", true
	default:
		return prefix + "\n\n" + suffix, true
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
