package app

import (
	"log/slog"
	"os"
	"os/signal"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/systemdata"
	"syscall"
)

type (
	RunSettings struct{}
	RunCommand  struct {
		RunSettings
	}
)

var (
	shuttingDown = make(chan os.Signal)
)

func startAppPump() {
	signal.Notify(shuttingDown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	slog.Debug("Starting signal handler")
	go func() {
		sigChan := <-shuttingDown
		slog.Info("Shutting down from signal", slog.String("signal", sigChan.String()))
		systemdata.StopSystemDataUpdates()
		rpc.Shutdown()
		os.Exit(0)
	}()
}
