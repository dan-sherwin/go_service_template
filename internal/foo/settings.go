package foo

import "github.com/dan-sherwin/go-app-settings"

type (
	Feebar struct{}
)

func (f Feebar) SettingName() string {
	return "feebar"
}

func (f Feebar) SettingDescription() string {
	return "The setting of feebar"
}

func (f Feebar) SettingSet(s string) error {
	feebar = s
	return nil
}

func (f Feebar) SettingGet() string {
	return feebar
}

func init() {
	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        "foobar",
		Description: "The setting of foobar",
		GetFunc: func() string {
			return foobar
		},
		SetFunc: func(s string) error {
			foobar = s
			return nil
		},
	})
	app_settings.RegisterSettingReceiver(&Feebar{})
}
