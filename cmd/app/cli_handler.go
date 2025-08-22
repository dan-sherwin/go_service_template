package app

import (
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/commands"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"github.com/alecthomas/kong"
	"go.corp.spacelink.com/sdks/go/app_settings"
)

var (
	CLICommand *kong.Context
	CLIConfig  struct {
		app_settings.SettingsDef
		commands.Commands
		Run RunCommand `cmd:"" help:"Run application in foreground"`
		ServiceDef
	}
	vars = kong.Vars{}
)

func processCLI() {
	vars["logging_level"] = LoggingLevel

	CLICommand = kong.Parse(&CLIConfig,
		kong.Name(consts.APPNAME),
		kong.Description(consts.APPNAME+" application"),
		kong.ShortUsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		vars,
	)
}
