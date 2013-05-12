package loglib

import (
	"github.com/stvp/go-toml-config"
)

var (
	TransportConfig = config.NewConfigSet("transportation settings", config.ExitOnError)

	twilioSid   = TransportConfig.String("twilio.sid", "")
	twilioToken = TransportConfig.String("twilio.token", "")

	smtpHostport = TransportConfig.String("smtp.hostport", ":25")
	smtpAuth     = TransportConfig.String("smtp.auth", "")

	FiltersConfig = config.NewConfigSet("filter settings", config.ExitOnError)
)

func LoadConfig(filters, transports string) error {
	var err error
	if err = FiltersConfig.Parse(filters); err != nil {
		return err
	}
	if err = TransportConfig.Parse(transports); err != nil {
		return err
	}
	return nil
}
