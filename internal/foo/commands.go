package foo

import (
	"fmt"
	"github.com/alecthomas/kong"
)

type (
	FooCommandDef struct {
		Foo FooCommand `cmd:"" help:"Foo command"`
	}
	FooCommand struct {
		Foobar string `default:"${fooFoobar}" help:"The setting of foobar"`
	}
)

func CommandVars() kong.Vars {
	return kong.Vars{
		"fooFoobar": foobar,
	}
}

func (f *FooCommand) Run() error {
	FOO()
	fmt.Printf("The setting of F.FOOBAR is %s\n", f.Foobar)
	return nil
}
