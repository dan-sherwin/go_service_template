# go_service_template — Go template with CLI + RPC + HTTP + systemd + TeamCity

Template repository URL: http://scm.dev.dsherwin.net/dsherwin/go_service_template

This repository is a reusable template for building small Go daemons/CLIs. It provides:
- CLI commands via Kong
- A Unix RPC server (net/rpc over a Unix domain socket)
- An HTTP server (via rest_api_server) you can extend with endpoints
- Structured logging with slog (Journald on Linux; Text to stdout on macOS)
- System data sampling (goroutine count, memory, CPU%)
- Systemd integration using takama/daemon
- Settings persistence via app_settings
- TeamCity CI setup, including an optional Deploy configuration via rsync/SSH
- A bootstrap tool to safely rename the module and app for a new project

On macOS, this template is intended for development; production artifacts target Linux.

## Quick start (using the bootstrap tool)

First and foremost, you need to run ```go mod tidy``` to initialize the module.

Run the bootstrap helper to safely rename the module and runtime app name.

Examples:
- Create a new app named your_app (sets both module path and APPNAME to your_app):
  go run ./dev/bootstrap your_app
- Preview planned changes without modifying files:
  go run ./dev/bootstrap your_app -dry-run

What the bootstrap tool does:
- Updates go.mod module path and rewrites Go imports that reference the old module path (AST-safe).
- Sets const APPNAME in cmd/app/consts/consts.go.
- Updates .teamcity/settings.kts:
  - param("app.name", "...")
  - the project description ("CI for ...")
  - ldflags import paths to match the new module
- Runs go mod tidy
- Optionally, it can be extended to update IDE files, but by default it does NOT touch .idea.

After bootstrapping:
1) Open the project in GoLand; it will re-index automatically.
2) Build and test:
   go build ./...
   go test -race ./...
3) Update any environment-specific settings (e.g., Deploy target in .teamcity/settings.kts) as needed.

## Building and running locally
- macOS/Linux (dev):
  go build -o ./dist/service_template ./cmd
  ./dist/service_template run

- Linux production build (as in TeamCity):
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags "-X 'scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts.Version=0.1.0'" -o ./dist/service_template ./cmd

The binary exposes a CLI with commands registered under cmd/app/commands. See internal/foo for examples of adding a command and a setting. To add a command, include the CommandDef inside app.commands.Commands and set defaults in app.Setup(), for example:

    utilities.MergeInto(vars, foo.CommandVars())

## Logging
- Linux: slog handler writes to journald via cmd/app/logger_linux.go
- macOS: slog uses TextHandler to stdout via cmd/app/logger_darwin.go
- Standard log keys: app, version, pid, user, component, error
- Set log level via CLI or settings (LOG_LEVEL env can override if you wire it)

## RPC
- Unix domain socket: /var/run/<APPNAME>.socket (0660 perms)
- Start server as part of daemon run path; client helpers dial per call and close

## HTTP
- rest_api_server started after daemon setup; add your own endpoints as needed
- Consider adding /healthz and /ready if useful

## Systemd integration
- Install/remove/start/stop/status via CLI under the Systemd command group
- Uses takama/daemon to register the service

## Versioning
- Build info is injected via -ldflags into cmd/app/consts (Version, Commit, BuildDate) and shown by the hidden buildinfo command

## TeamCity CI/CD
- .teamcity/settings.kts contains a Build configuration:
  - go mod tidy, go vet, go test -race, and a Linux/amd64 build with ldflags
- It also contains an optional Deploy configuration using rsync/SSH to a target host and a systemctl restart. Parameters to set per environment:
  - deploy.dest_user (default dsherwin)
  - deploy.dest_host (e.g., monitor1.corp.spacelink.com)
  - deploy.dest_path (default /usr/local/%app.name%/%app.name%)
  - service.name (defaults to %app.name%)

## Template notes
- Keep runtime app name centralized in cmd/app/consts/consts.go (APPNAME), default is "service_template".
- An example package exists at internal/foo to demonstrate settings and commands integration. To remove it, delete the internal/foo directory and remove any references to it:
  - app.Setup(): remove utilities.MergeInto(vars, foo.CommandVars())
  - cmd/app/commands/commands.go: remove foo.FooCommandDef from the Commands struct
  - Any imports referencing internal/foo
- Prefer the bootstrap tool to rename this template rather than search/replace.
- See .junie/guidelines.md for project conventions.


