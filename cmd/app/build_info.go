package app

import (
	"fmt"
	"runtime/debug"
	"strings"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

type (
	BuildInfoCommandDef struct {
		Buildinfo BuildInfoCommand `cmd:"" hidden:"" help:"Show the build information"`
	}
	BuildInfoCommand struct{}
)

func (b *BuildInfoCommand) Run() error {
	if bi, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("\nApp Name: %s\nGo Version: %s\nApp Version: %s\nCommit: %s\nBuildDate: %s\nPath: %s\nModuleVersion: %s\n", consts.APPNAME, bi.GoVersion, consts.Version, consts.Commit, consts.BuildDate, bi.Path, bi.Main.Version)
		for _, s := range bi.Settings {
			if strings.HasPrefix(s.Key, "-") {
				continue
			}
			fmt.Printf("%s: %s\n", s.Key, s.Value)
		}
		fmt.Printf("\n\n")
	} else {
		fmt.Println("no build information available")
	}
	return nil
}
