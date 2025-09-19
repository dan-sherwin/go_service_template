package app

import (
	"fmt"
	"github.com/takama/daemon"
	"log/slog"
	"os"
	"runtime"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

type (
	SystemService struct {
		daemon.Daemon
	}
	InstallServiceCommand struct{}
	RemoveServiceCommand  struct{}
	StartServiceCommand   struct{}
	StopServiceCommand    struct{}
	RestartServiceCommand struct{}
	ServiceStatusCommand  struct{}
	Service               struct {
		Install InstallServiceCommand `cmd:"" group:"Systemd" help:"Install the systemdService as a systemd systemdService"`
		Remove  RemoveServiceCommand  `cmd:"" group:"Systemd" help:"Remove the systemdService from systemd"`
		Start   StartServiceCommand   `cmd:"" group:"Systemd" help:"Start the systemdService"`
		Stop    StopServiceCommand    `cmd:"" group:"Systemd" help:"Stop the systemdService"`
		Restart RestartServiceCommand `cmd:"" group:"Systemd" help:"Restart the systemdService"`
		Status  ServiceStatusCommand  `cmd:"" group:"Systemd" help:"Show the status of the systemdService" default:"1"`
	}
	ServiceDef struct {
		Service Service `cmd:"" help:"Service management commands" name:"systemd"`
	}
)

var (
	systemdService *SystemService
)

func setupSystemdService() {
	var srv daemon.Daemon
	var err error
	var kind daemon.Kind
	if runtime.GOOS == "darwin" {
		kind = daemon.GlobalDaemon
	} else {
		kind = daemon.SystemDaemon
	}
	srv, err = daemon.New(consts.APPNAME, "", kind)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	systemdService = &SystemService{srv}
}

func (i *InstallServiceCommand) Run() error {
	slog.Info("systemd install requested")
	if status, err := systemdService.Install("run"); err != nil {
		slog.Error("systemd install failed", slog.String("error", err.Error()))
		return err
	} else {
		fmt.Println(status)
		slog.Info("systemd install", slog.String("status", status))
	}
	return nil
}

func (r *RemoveServiceCommand) Run() error {
	slog.Info("systemd remove requested")
	if status, err := systemdService.Remove(); err != nil {
		slog.Error("systemd remove failed", slog.String("error", err.Error()))
		return err
	} else {
		fmt.Println(status)
		slog.Info("systemd remove", slog.String("status", status))
	}
	return nil
}

func (s *StartServiceCommand) Run() error {
	slog.Info("systemd start requested")
	if status, err := systemdService.Start(); err != nil {
		slog.Error("systemd start failed", slog.String("error", err.Error()))
		return err
	} else {
		fmt.Println(status)
		slog.Info("systemd start", slog.String("status", status))
	}
	return nil
}

func (k *StopServiceCommand) Run() error {
	slog.Info("systemd stop requested")
	if status, err := systemdService.Stop(); err != nil {
		slog.Error("systemd stop failed", slog.String("error", err.Error()))
		return err
	} else {
		fmt.Println(status)
		slog.Info("systemd stop", slog.String("status", status))
	}
	return nil
}

func (r *RestartServiceCommand) Run() error {
	slog.Info("systemd restart requested")
	if status, err := systemdService.ReStart(); err != nil {
		slog.Error("systemd restart failed", slog.String("error", err.Error()))
		return err
	} else {
		for _, s := range status {
			fmt.Println(s)
		}
		slog.Info("systemd restart", slog.Any("status", status))
	}
	return nil
}

func (s *ServiceStatusCommand) Run() error {
	slog.Info("systemd status requested")
	if status, err := systemdService.Status(); err != nil {
		slog.Error("systemd status failed", slog.String("error", err.Error()))
		return err
	} else {
		fmt.Println(status)
		slog.Info("systemd status", slog.String("status", status))
	}
	return nil
}

func (s *SystemService) ReStart() (statuses []string, err error) {
	var status string
	status, err = s.Daemon.Stop()
	if err != nil {
		return
	}
	statuses = append(statuses, status)
	status, err = s.Daemon.Start()
	if err != nil {
		return
	}
	statuses = append(statuses, status)
	return
}
