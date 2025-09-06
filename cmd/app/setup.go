package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/systemdata"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/internal/foo"

	"go.corp.spacelink.com/sdks/go/app_settings"
	"go.corp.spacelink.com/sdks/go/rest_api_server"
	"go.corp.spacelink.com/sdks/go/utilities"
)

func init() {
	app_settings.RegisterSetting(&app_settings.Setting{
		SetFunc: func(s string) error {
			if s == "" {
				return fmt.Errorf("HTTP Listening Address cannot be empty")
			}
			rest_api_server.ListeningAddress = s
			return nil
		},
		GetFunc: func() string {
			return rest_api_server.ListeningAddress
		},
		Name:        "http_listening_address",
		Description: "HTTP Listening address",
	})
}

func Setup() {
	setWorkingDir()
	slog.Debug("working directory set")
	slog.Info("initializing settings DB", slog.String("app", consts.APPNAME))
	app_settings.Setup(consts.APPNAME+".db", app_settings.SettingsOptions{
		RpcSocketPathToListRunningSettings: rpc.SocketPath,
		KongVars:                           &vars,
	})
	utilities.MergeInto(vars, foo.CommandVars())
	processCLI()
	LoggingLevel = cliConfig.Logging.Level
	initLogger()
	slog.Info("logger initialized")
	setupSystemdService()
}

func SetupDaemon() {
	slog.Debug(consts.APPNAME + " app daemon setup")
	// Signals handled in startAppPump; avoid SIGKILL which cannot be trapped
	signal.Notify(shuttingDown, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	systemdata.StartSystemDataUpdates()
	startAppPump()
	rpc.StartServer()
	slog.Info("daemon setup complete")
}

func setWorkingDir() {
	ex, err := os.Executable()
	if err != nil {
		slog.Error("cannot resolve executable path", slog.String("error", err.Error()))
		os.Exit(1)
	}
	exPath := filepath.Dir(ex)
	if err := os.Chdir(exPath); err != nil {
		slog.Error("chdir failed", slog.String("path", exPath), slog.String("error", err.Error()))
		os.Exit(1)
	}
}
