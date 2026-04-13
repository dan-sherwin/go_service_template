#!/usr/bin/env bash
set -euo pipefail
trap "rm -f coverage.out" EXIT

export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.2}"

# Ensure tools exist
command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest

# 1) Tidy and ensure no changes
go mod tidy

# 2) Build, Vet, Test (race)
go build ./...
go vet ./...
go test ./... -race -count=1 -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out \
| awk -v thr="${COVER_THRESH:-0}" '
/^total:/ {
  gsub(/%/, "", $3);    # strip percent sign
  cov = $3 + 0;
  if (cov < thr) {
    printf "FAIL: coverage %.1f%% < %d%%\n", cov, thr;
    exit 1
  } else {
    exit 0
  }
}
END {
  # If we never saw a total line, treat as failure
  if (NR == 0) { print "ERROR: no coverage data."; exit 2 }
}'
rm -f coverage.out

# 3) Lint
GOGC=off golangci-lint config verify
GOGC=off golangci-lint run --timeout 5m

# 4) Vulnerabilities
govulncheck -test ./...

# 5) gofmt test
test -z "$(gofmt -s -l .)" || { echo "gofmt needed"; exit 1; }
