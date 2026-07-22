#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

backup_sqlite_database() {
	local source="$1"
	local destination="$2"
	local backup_path integrity
	if ! command -v sqlite3 >/dev/null 2>&1; then
		printf 'sqlite3 is required to safely migrate %s to %s\n' "$source" "$destination" >&2
		return 2
	fi
	backup_path="$(mktemp "${destination}.backup.XXXXXX")"
	if ! sqlite3 "$source" '.timeout 5000' ".backup '$backup_path'"; then
		rm -f -- "$backup_path"
		printf 'Failed to create a consistent SQLite backup from %s\n' "$source" >&2
		return 1
	fi
	if ! integrity="$(sqlite3 "$backup_path" 'PRAGMA quick_check;')" || [[ "$integrity" != "ok" ]]; then
		rm -f -- "$backup_path"
		printf 'SQLite backup validation failed for %s\n' "$source" >&2
		return 1
	fi
	chmod 0600 "$backup_path"
	mv "$backup_path" "$destination"
}

app_name="service_template"
dev_dir="build/dev"
legacy_database="build/$app_name.db"
dev_database="$dev_dir/$app_name.db"
host_os="$(go env GOHOSTOS)"
host_arch="$(go env GOHOSTARCH)"

mkdir -p "$dev_dir"
if [[ -f "$legacy_database" && ! -e "$dev_database" ]]; then
	backup_sqlite_database "$legacy_database" "$dev_database"
	printf 'Copied existing development settings to %s\n' "$dev_database"
fi

GOOS="$host_os" GOARCH="$host_arch" go build -o "$dev_dir/$app_name" ./cmd
printf 'Built %s\n' "$dev_dir/$app_name"
