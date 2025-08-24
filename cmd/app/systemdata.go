package app

import (
	"fmt"

	u "github.com/bcicen/go-units"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/systemdata"
)

type (
	SystemDataCommandDef struct {
		Systemdata SystemDataCommand `cmd:"" help:"Show the system data"`
	}
	SystemDataCommand struct{}
)

func init() {
	rpc.RegisterName("SystemData", &SystemDataCommand{})
}

func (f *SystemDataCommand) Run() error {
	var s string
	err := rpc.Call("SystemData.GetSystemData", &struct{}{}, &s)
	if err != nil {
		return err
	}
	fmt.Println(s)
	return nil
}

func (f *SystemDataCommand) GetSystemData(_ *struct{}, data *string) error {
	opts := u.FmtOptions{
		Label:     true, // append unit name/symbol
		Short:     true, // use unit symbol
		Precision: 2,
	}
	sd := systemdata.GetSystemData()

	*data = fmt.Sprintf(systemDataFormat,
		u.NewValue(float64(sd.Alloc), u.Byte).MustConvert(u.MegaByte).Fmt(opts),
		u.NewValue(float64(sd.SystemAlloc), u.Byte).MustConvert(u.MegaByte).Fmt(opts),
		sd.NumGoRoutines,
		sd.NumCPUs,
		sd.CPUPercent)
	return nil
}

const systemDataFormat = `
Alloc: %s
SystemAlloc: %s
NumGoRoutines: %d
NumCPUs: %d
CPUPercent: %.1f
`
