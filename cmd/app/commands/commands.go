package commands

import (
	"scm.dev.dsherwin.net/dsherwin/go_service_template/internal/foo"
)

type (
	Commands struct {
		BuildInfoCommandDef
		SystemDataCommandDef
		foo.FooCommandDef
	}
)
