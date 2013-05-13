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

	esUrl = TransportConfig.String("elasticsearch.url", ":9200")
	esTTL = TransportConfig.Int("elasticsearch.ttl", 90)

	FiltersConfig = config.NewConfigSet("filter settings", config.ExitOnError)
)

func LoadConfig(transports, filters string) error {
	var err error
	if err = TransportConfig.Parse(transports); err != nil {
		return err
	}
	if *esUrl != "" {
		StoreCh = make(chan *Message)
		go storeEs(*esUrl, *esTTL)
	}
	if err = FiltersConfig.Parse(filters); err != nil {
		return err
	}
	return nil
}
