package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgsSupportsDryRunBeforeAppName(t *testing.T) {
	dryRun, appName, err := parseArgs([]string{"-dry-run", "chronix_tester_platform"})
	if err != nil {
		t.Fatalf("parse args returned error: %v", err)
	}
	if !dryRun {
		t.Fatalf("expected dryRun to be true")
	}
	if appName != "chronix_tester_platform" {
		t.Fatalf("unexpected app name: got %q", appName)
	}
}

func TestParseArgsSupportsDryRunAfterAppName(t *testing.T) {
	dryRun, appName, err := parseArgs([]string{"chronix_tester_platform", "-dry-run"})
	if err != nil {
		t.Fatalf("parse args returned error: %v", err)
	}
	if !dryRun {
		t.Fatalf("expected dryRun to be true")
	}
	if appName != "chronix_tester_platform" {
		t.Fatalf("unexpected app name: got %q", appName)
	}
}

func TestParseArgsRejectsUnknownFlags(t *testing.T) {
	_, _, err := parseArgs([]string{"--wat", "chronix_tester_platform"})
	if err == nil {
		t.Fatalf("expected error for unknown flag")
	}
}

func TestUpdateRunConfigurationsUsesDevelopmentBuildDirectory(t *testing.T) {
	root := t.TempDir()
	runConfigDir := filepath.Join(root, "dev", "runConfigurations")
	if err := os.MkdirAll(runConfigDir, 0o755); err != nil {
		t.Fatalf("create run configuration directory: %v", err)
	}
	path := filepath.Join(runConfigDir, "go build main.go.run.xml")
	content := `<component>
  <module name="go_service_template" />
  <go_parameters value="-o $ProjectFileDir$/build/dev/service_template" />
  <package value="scm.dev.dsherwin.net/dsherwin/go_service_template" />
  <output_directory value="$PROJECT_DIR$/build/dev" />
</component>`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write run configuration: %v", err)
	}

	updateRunConfigurations(
		root,
		"scm.dev.dsherwin.net/dsherwin/go_service_template",
		"example_service",
		"example_service",
	)

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read run configuration: %v", err)
	}
	if !strings.Contains(string(updated), "build/dev/example_service") {
		t.Fatalf("run configuration did not use renamed development binary: %s", updated)
	}
}

func TestUpdateReadmeSeparatesDevelopmentAndProductionBuilds(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("template"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	updateReadme(
		root,
		"scm.dev.dsherwin.net/dsherwin/go_service_template",
		"example_service",
		"example_service",
	)

	updated, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(updated)
	for _, expected := range []string{
		"./dev/build-dev.sh",
		"./build/dev/example_service run",
		"-o ./dist/example_service ./cmd",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated README is missing %q", expected)
		}
	}
}

func TestUpdateDevBuildScriptRenamesApplication(t *testing.T) {
	root := t.TempDir()
	devDir := filepath.Join(root, "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("create dev directory: %v", err)
	}
	path := filepath.Join(devDir, "build-dev.sh")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\napp_name=\"service_template\"\n"), 0o755); err != nil {
		t.Fatalf("write development build script: %v", err)
	}

	updateDevBuildScript(root, "example_service")

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read development build script: %v", err)
	}
	if !strings.Contains(string(updated), `app_name="example_service"`) {
		t.Fatalf("development build script did not receive app name: %s", updated)
	}
}
