package loglib

import (
	"github.com/pelletier/go-toml"
	"github.com/stvp/go-toml-config"
	"log"
)

var (
	TransportConfig = config.NewConfigSet("transportation settings", config.ExitOnError)
	gelfUdpPort     = TransportConfig.Int("gelf.udp", 12201)

	twilioSid   = TransportConfig.String("twilio.sid", "")
	twilioToken = TransportConfig.String("twilio.token", "")

	smtpHostport = TransportConfig.String("smtp.hostport", ":25")
	smtpAuth     = TransportConfig.String("smtp.auth", "")

	esUrl = TransportConfig.String("elasticsearch.url", ":9200")
	esTTL = TransportConfig.Int("elasticsearch.ttl", 90)

	Alerters map[string]Alerter
	Matchers map[string]Matcher
	Rules    []Rule
)

func LoadConfig(transports, filters string) error {
	var err error
	log.Printf("loading transports config file %s", filters)
	if err = TransportConfig.Parse(transports); err != nil {
		return err
	}
	if *esUrl != "" {
		log.Printf("starting storage goroutine for %s", *esUrl)
		StoreCh = make(chan *Message)
		go storeEs(*esUrl, *esTTL)
	}
	log.Printf("loading filters config file %s", filters)
	tree, e := toml.LoadFile(filters)
	if e != nil {
		return e
	}
	log.Printf("building matchers from %s", tree.Get("filters"))
	if Matchers, err = BuildMatchers(tree); err != nil {
		return err
	}
	log.Printf("matchers: %v", Matchers)

	log.Printf("building destinations from %s", tree.Get("destinations"))
	if Alerters, err = BuildAlerters(tree); err != nil {
		return err
	}
	log.Printf("alerters: %v", Alerters)

	log.Printf("building rules")
	if Rules, err = BuildRules(tree, Matchers, Alerters); err != nil {
		return err
	}
	log.Printf("rules: %v", Rules)

	return nil
}
