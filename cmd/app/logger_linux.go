package app

import (
	"log/slog"
	"os"
	user2 "os/user"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

var LoggingLevel = "debug"

func initLogger() {
	logger := slog.New(&JournaldHandler{})

	user, _ := user2.Current()
	child := logger.With(
		slog.Int("pid", os.Getpid()),
		slog.String("user", user.Username),
		slog.String("app", consts.APPNAME),
		slog.String("version", consts.Version),
	)
	slog.SetDefault(child)
}
