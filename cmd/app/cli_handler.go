package app

import (
	"os"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"github.com/alecthomas/kong"
	"github.com/willabides/kongplete"
	"go.corp.spacelink.com/sdks/go/app_settings"
)

type (
	CLIConfig struct {
		app_settings.SettingsDef
		Commands
		Run RunCommand `cmd:"" help:"Run application in foreground"`
		ServiceDef
		InstallCompletions kongplete.InstallCompletions `cmd:"" name:"completionscript" help:"Install shell completions (bash|zsh|fish)." hidden:""`
	}
)

var (
	CLICommand *kong.Context
	cliConfig  CLIConfig
	vars       = kong.Vars{}
)

func processCLI() {
	vars["logging_level"] = LoggingLevel
	parser := kong.Must(&cliConfig,
		kong.Name(consts.APPNAME),
		kong.Description(consts.APPNAME+" application"),
		kong.ShortUsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		vars,
	)

	kongplete.Complete(parser)
	var err error
	CLICommand, err = parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	postParseProcessing(CLICommand, &cliConfig)

}
