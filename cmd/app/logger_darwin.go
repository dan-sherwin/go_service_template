package app

import (
	"log/slog"
	"os"
	user2 "os/user"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

var LoggingLevel = "debug"

func initLogger() {
	lvl := slog.LevelDebug
	if LoggingLevel == "info" {
		lvl = slog.LevelInfo
	} else if LoggingLevel == "warn" {
		lvl = slog.LevelWarn
	} else if LoggingLevel == "error" {
		lvl = slog.LevelError
	}
	opts := &slog.HandlerOptions{
		Level: lvl,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	user, _ := user2.Current()
	child := logger.With(
		slog.Int("pid", os.Getpid()),
		slog.String("user", user.Username),
		slog.String("app", consts.APPNAME),
		slog.String("version", consts.Version),
	)
	slog.SetDefault(child)
}
