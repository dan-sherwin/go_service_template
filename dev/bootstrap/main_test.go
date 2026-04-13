package main

import "testing"

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
