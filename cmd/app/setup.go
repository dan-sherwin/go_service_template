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
	// Logging level setting
	app_settings.RegisterSetting(&app_settings.Setting{
		SetFunc: func(s string) error {
			if s == "" {
				return fmt.Errorf("log level cannot be empty")
			}
			LoggingLevel = s
			initLogger()
			return nil
		},
		GetFunc:     func() string { return LoggingLevel },
		Name:        "log_level",
		Description: "Logging level (debug|info|warn|error)",
	})
	// RPC socket path setting
	app_settings.RegisterSetting(&app_settings.Setting{
		SetFunc: func(s string) error {
			if s == "" {
				return fmt.Errorf("rpc socket path cannot be empty")
			}
			rpc.SocketPath = s
			return nil
		},
		GetFunc:     func() string { return rpc.SocketPath },
		Name:        "rpc_socket_path",
		Description: "Path to Unix domain socket for RPC",
	})
}

func Setup() {
	setWorkingDir()
	slog.Debug("working directory set")
	app_settings.Setup(consts.APPNAME+".db", app_settings.SettingsOptions{
		RpcSocketPathToListRunningSettings: rpc.SocketPath,
		KongVars:                           &vars,
	})
	utilities.MergeInto(vars, foo.CommandVars())
	processCLI()
	LoggingLevel = cliConfig.Logging.Level
	initLogger()
	slog.Info("build info", slog.String("version", consts.Version), slog.String("commit", consts.Commit), slog.String("build_date", consts.BuildDate))
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
