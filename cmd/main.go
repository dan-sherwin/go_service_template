package main

import (
	"fmt"
	"log/slog"
	"os"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"github.com/dan-sherwin/go-rest-api-server"
	"github.com/dan-sherwin/go-utilities"
)

func main() {
	app.StartRecorder()
	defer app.PanicDefer()
	app.Setup()
	processCommand()
	slog.Info("Run command called. Starting "+consts.APPNAME+" as a daemon.",
		slog.String("version", consts.Version),
		slog.String("commit", consts.Commit),
		slog.String("buildDate", consts.BuildDate),
	)
	if utilities.DaemonAlreadyRunning(consts.APPNAME) {
		fmt.Println("Daemon already running. Exiting.")
		return
	}
	app.SetupDaemon()
	restapi.StartHttpServer()
	slog.Info("http server started", slog.String("addr", restapi.ListeningAddress))
	slog.Info(consts.APPNAME + " is running.")
	app.WaitForShutdown()
	if err := restapi.ShutdownHttpServer(); err != nil {
		slog.Error("Failed to shutdown HTTP server", slog.String("error", err.Error()))
	}
	slog.Info(consts.APPNAME + " stopped.")
}

func processCommand() {
	if app.CLICommand.Command() == "run" {
		return
	}
	slog.Info("Command called", "command", app.CLICommand.Command())
	if err := app.CLICommand.Run(); err != nil {
		slog.Error("Error running command", "error", err)
	}
	os.Exit(0)
}
