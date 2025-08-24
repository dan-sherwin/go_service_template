package app

import (
	"github.com/alecthomas/kong"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/internal/foo"
)

type (
	Commands struct {
		BuildInfoCommandDef
		SystemDataCommandDef
		foo.FooCommandDef
	}
)

func postParseProcessing(cliCommand *kong.Context, cliConfig *CLIConfig) {
}
