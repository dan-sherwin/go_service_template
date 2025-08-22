package main

import (
	"fmt"
	"log/slog"
	"os"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"go.corp.spacelink.com/sdks/go/rest_api_server"
	"go.corp.spacelink.com/sdks/go/utilities"
)

func main() {
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
	rest_api_server.StartHttpServer()
	slog.Info("http server started", slog.String("addr", rest_api_server.ListeningAddress))
	slog.Info(consts.APPNAME + " is running.")
	select {}
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
