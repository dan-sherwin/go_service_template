# Project Guidelines

## Logging
- Use slog with structured keys. Standard keys: app, version, pid, user, component, error.
- Levels: debug for diagnostics, info for lifecycle, warn for recoverable issues, error for failures.
- Linux: default to JournaldHandler; macOS: TextHandler to stdout.

## Error Handling
- Avoid panics in production paths. Log and return or exit with non-zero status.
- Wrap errors with context using fmt.Errorf("context: %w", err).
- Use key name "error" consistently for error fields.

## Configuration
- Precedence: CLI flags > environment variables > persisted app_settings > defaults.
- Common envs: LOG_LEVEL, HTTP_LISTEN_ADDR, RPC_SOCKET_PATH.

## Contexts and Shutdown
- Use context or explicit Stop/Shutdown methods for long-running goroutines.
- Trap SIGINT, SIGTERM, SIGQUIT, SIGHUP; SIGKILL cannot be trapped.
- On shutdown: stop tickers/workers, close RPC listener, and shutdown HTTP server if supported.

## RPC
- Prefer Unix domain sockets by default; make path configurable for non-root users (e.g., XDG_RUNTIME_DIR). Default permissions 0660.
- Ensure client connections are closed or use rpc.Call helper which dials and closes per call.
- Validate inputs and return typed errors.

## HTTP
- Start server after RPC is ready. Provide /healthz and /ready endpoints if applicable in rest_api_server.
- Consider enabling JSON logs in production for log aggregation.

## Dependencies
- Pin major.minor Go version in go.mod (e.g., `go 1.24`). Run `go mod tidy` and `go vet` in CI.
- Prefer `golangci-lint` locally and in CI.

## Versioning
- Inject Version, Commit, BuildDate via `-ldflags`. Log them at startup and expose via a CLI command.

## Testing
- Unit tests for RPC handlers and helpers. Use race detector. Avoid tests depending on /var/run unless using temp dirs.

## Security
- Restrict socket permissions to 0660 and (optionally) set group ownership.
- Avoid logging secrets; redact sensitive fields.

## Style
- Follow Go naming conventions; unexport local vars unless needed.
- Keep consistent slog key names and avoid inconsistent key spellings.
