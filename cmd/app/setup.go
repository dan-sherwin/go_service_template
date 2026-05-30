package app

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/systemdata"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/internal/foo"

	appruntime "github.com/dan-sherwin/go-app-runtime"
	"github.com/dan-sherwin/go-app-settings"
	"github.com/dan-sherwin/go-applog"
	"github.com/dan-sherwin/go-rest-api-server"
	"github.com/dan-sherwin/go-utilities"
)

func init() {
	appruntime.Setup(appruntime.SetupOptions{
		AppName:     consts.APPNAME,
		Version:     consts.Version,
		Commit:      consts.Commit,
		BuildDate:   consts.BuildDate,
		RegisterRPC: rpc.RegisterName,
		CallRPC:     rpc.Call,
	})
	app_settings.RegisterSetting(&app_settings.Setting{
		SetFunc: func(s string) error {
			if s == "" {
				return fmt.Errorf("HTTP Listening Address cannot be empty")
			}
			restapi.ListeningAddress = s
			return nil
		},
		GetFunc: func() string {
			return restapi.ListeningAddress
		},
		Name:        "http_listening_address",
		Description: "HTTP Listening address",
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
	applog.Debug("working directory set")
	if err := app_settings.Setup(consts.APPNAME+".db", app_settings.SettingsOptions{
		RpcSocketPathToListRunningSettings: rpc.SocketPath,
		KongVars:                           &vars,
	}); err != nil {
		applog.Error("Failed to setup settings", applog.String("error", err.Error()))
		os.Exit(1)
	}
	appruntime.ConfigureKongVars(&vars)
	utilities.MergeInto(vars, foo.CommandVars())
	processCLI()
	if err := appruntime.SetLevel(cliConfig.Logging.Level); err != nil {
		applog.Error("Failed to apply logging level", applog.String("level", cliConfig.Logging.Level), applog.String("error", err.Error()))
		os.Exit(1)
	}
	appruntime.SetVerbose(cliConfig.Verbose)
	applog.Info("build info", applog.String("version", consts.Version), applog.String("commit", consts.Commit), applog.String("buildDate", consts.BuildDate))
	setupSystemdService()
}

func SetupDaemon() {
	applog.Debug(consts.APPNAME + " app daemon setup")
	// Signals handled in startAppPump; avoid SIGKILL which cannot be trapped
	signal.Notify(shutdownSignals, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	systemdata.StartSystemDataUpdates()
	startAppPump()
	if err := rpc.StartServer(); err != nil {
		applog.Error("Failed to start RPC server", applog.String("error", err.Error()))
		os.Exit(1)
	}
	applog.Info("daemon setup complete")
}

func setWorkingDir() {
	ex, err := os.Executable()
	if err != nil {
		applog.Error("cannot resolve executable path", applog.String("error", err.Error()))
		os.Exit(1)
	}
	exPath := filepath.Dir(ex)
	if err := os.Chdir(exPath); err != nil {
		applog.Error("chdir failed", applog.String("path", exPath), applog.String("error", err.Error()))
		os.Exit(1)
	}
}
