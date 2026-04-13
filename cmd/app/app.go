package app

import (
	"log/slog"
	"os"
	"os/signal"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/systemdata"
	"sync"
)

type (
	RunSettings struct{}
	RunCommand  struct {
		RunSettings
	}
)

var (
	shutdownSignals = make(chan os.Signal, 1)
	shutdownDone    = make(chan struct{})
	shutdownOnce    sync.Once
)

func startAppPump() {
	slog.Debug("Starting signal handler")
	go func() {
		sigChan := <-shutdownSignals
		slog.Info("Shutting down from signal", slog.String("signal", sigChan.String()))
		shutdown()
	}()
}

func shutdown() {
	shutdownOnce.Do(func() {
		signal.Stop(shutdownSignals)
		systemdata.StopSystemDataUpdates()
		rpc.Shutdown()
		close(shutdownDone)
	})
}

func WaitForShutdown() {
	<-shutdownDone
}
