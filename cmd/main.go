package main

import (
	"fmt"
	"github.com/dan-sherwin/go-applog"
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
	applog.Info("Run command called. Starting "+consts.APPNAME+" as a daemon.",
		applog.String("version", consts.Version),
		applog.String("commit", consts.Commit),
		applog.String("buildDate", consts.BuildDate),
	)
	if utilities.DaemonAlreadyRunning(consts.APPNAME) {
		fmt.Println("Daemon already running. Exiting.")
		return
	}
	app.SetupDaemon()
	restapi.StartHttpServer()
	applog.Info("http server started", applog.String("addr", restapi.ListeningAddress))
	applog.Info(consts.APPNAME + " is running.")
	app.WaitForShutdown()
	if err := restapi.ShutdownHttpServer(); err != nil {
		applog.Error("Failed to shutdown HTTP server", applog.String("error", err.Error()))
	}
	applog.Info(consts.APPNAME + " stopped.")
}

func processCommand() {
	if app.CLICommand.Command() == "run" {
		return
	}
	applog.Info("Command called", "command", app.CLICommand.Command())
	if err := app.CLICommand.Run(); err != nil {
		applog.Error("Error running command", "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
