package app

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/takama/daemon"
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
	status, err := systemdService.Install("run")
	if err != nil {
		slog.Error("systemd install failed", slog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	slog.Info("systemd install", slog.String("status", status))
	return nil
}

func (r *RemoveServiceCommand) Run() error {
	slog.Info("systemd remove requested")
	status, err := systemdService.Remove()
	if err != nil {
		slog.Error("systemd remove failed", slog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	slog.Info("systemd remove", slog.String("status", status))
	return nil
}

func (s *StartServiceCommand) Run() error {
	slog.Info("systemd start requested")
	status, err := systemdService.Start()
	if err != nil {
		slog.Error("systemd start failed", slog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	slog.Info("systemd start", slog.String("status", status))
	return nil
}

func (k *StopServiceCommand) Run() error {
	slog.Info("systemd stop requested")
	status, err := systemdService.Stop()
	if err != nil {
		slog.Error("systemd stop failed", slog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	slog.Info("systemd stop", slog.String("status", status))
	return nil
}

func (r *RestartServiceCommand) Run() error {
	slog.Info("systemd restart requested")
	status, err := systemdService.ReStart()
	if err != nil {
		slog.Error("systemd restart failed", slog.String("error", err.Error()))
		return err
	}
	for _, s := range status {
		fmt.Println(s)
	}
	slog.Info("systemd restart", slog.Any("status", status))
	return nil
}

func (s *ServiceStatusCommand) Run() error {
	slog.Info("systemd status requested")
	status, err := systemdService.Status()
	if err != nil {
		slog.Error("systemd status failed", slog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	slog.Info("systemd status", slog.String("status", status))
	return nil
}

func (s *SystemService) ReStart() (statuses []string, err error) {
	var status string
	status, err = s.Stop()
	if err != nil {
		return
	}
	statuses = append(statuses, status)
	status, err = s.Start()
	if err != nil {
		return
	}
	statuses = append(statuses, status)
	return
}
