package app

import (
	"fmt"
	"github.com/dan-sherwin/go-applog"
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
	applog.Info("systemd install requested")
	status, err := systemdService.Install("run")
	if err != nil {
		applog.Error("systemd install failed", applog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	applog.Info("systemd install", applog.String("status", status))
	return nil
}

func (r *RemoveServiceCommand) Run() error {
	applog.Info("systemd remove requested")
	status, err := systemdService.Remove()
	if err != nil {
		applog.Error("systemd remove failed", applog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	applog.Info("systemd remove", applog.String("status", status))
	return nil
}

func (s *StartServiceCommand) Run() error {
	applog.Info("systemd start requested")
	status, err := systemdService.Start()
	if err != nil {
		applog.Error("systemd start failed", applog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	applog.Info("systemd start", applog.String("status", status))
	return nil
}

func (k *StopServiceCommand) Run() error {
	applog.Info("systemd stop requested")
	status, err := systemdService.Stop()
	if err != nil {
		applog.Error("systemd stop failed", applog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	applog.Info("systemd stop", applog.String("status", status))
	return nil
}

func (r *RestartServiceCommand) Run() error {
	applog.Info("systemd restart requested")
	status, err := systemdService.ReStart()
	if err != nil {
		applog.Error("systemd restart failed", applog.String("error", err.Error()))
		return err
	}
	for _, s := range status {
		fmt.Println(s)
	}
	applog.Info("systemd restart", applog.Any("status", status))
	return nil
}

func (s *ServiceStatusCommand) Run() error {
	applog.Info("systemd status requested")
	status, err := systemdService.Status()
	if err != nil {
		applog.Error("systemd status failed", applog.String("error", err.Error()))
		return err
	}
	fmt.Println(status)
	applog.Info("systemd status", applog.String("status", status))
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
