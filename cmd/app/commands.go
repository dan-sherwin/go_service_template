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
		RecorderCommandDef
		Completions CompletionsCommandDef `cmd:"" name:"autoCompletions" help:"Manage shell completions"`
	}
)

func postParseProcessing(_ *kong.Context, _ *CLIConfig) {
}
