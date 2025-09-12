package app

import (
	"log/slog"
	"os"
	"os/user"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setDefaultLogger(base *slog.Logger) {
	user, _ := user.Current()
	child := base.With(
		slog.Int("pid", os.Getpid()),
		slog.String("user", user.Username),
		slog.String("app", consts.APPNAME),
		slog.String("version", Version),
		slog.String("commit", Commit),
		slog.String("build_date", BuildDate),
	)
	slog.SetDefault(child)
}
