package loglib

import (
	"github.com/pelletier/go-toml"
	"github.com/stvp/go-toml-config"
	"log"
)

var (
	TransportConfig = config.NewConfigSet("transportation settings", config.ExitOnError)
	from            = TransportConfig.String("from", "woodchuck")
	gelfUdpPort     = TransportConfig.Int("gelf.udp", 12201)
	gelfTcpPort     = TransportConfig.Int("gelf.tcp", 0)
	gelfHttpPort    = TransportConfig.Int("gelf.http", 0)

	twilioSid   = TransportConfig.String("twilio.sid", "")
	twilioToken = TransportConfig.String("twilio.token", "")

	smtpHostport = TransportConfig.String("smtp.hostport", ":25")
	smtpAuth     = TransportConfig.String("smtp.auth", "")

	esUrl = TransportConfig.String("elasticsearch.url", "http://localhost:9200")
	esTTL = TransportConfig.Int("elasticsearch.ttl", 90)
)

type SMSSender interface {
	Send(to, message string) error
}
type EmailSender interface {
	Send(to, subject, body string) error
}

type Server struct {
	in, store chan *Message
	SMS       SMSSender
	Email     EmailSender
	Rules     []Rule
	Matchers  map[string]Matcher
	Alerters  map[string]Alerter
	routines  []func()
}

func LoadConfig(transports, filters string) (s *Server, err error) {
	log.Printf("loading transports config file %s", filters)
	if err = TransportConfig.Parse(transports); err != nil {
		return
	}
    s = &Server{routines: make([]func(), 0, 4), in: make(chan *Message)}
	if *esUrl != "" {
		log.Printf("starting storage goroutine for %s", *esUrl)
		s.store = make(chan *Message)
		s.routines = append(s.routines, func() {
			storeEs(*esUrl, *esTTL, s.store)
		})
	}
	if *twilioSid != "" {
		s.SMS = NewTwilio(*from, *twilioSid, *twilioToken)
	}
	if *smtpHostport != "" {
		s.Email = NewEmailSender(*from, *smtpHostport, *smtpAuth)
	}
	if *gelfUdpPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfUdp(*gelfUdpPort, s.in)
		})
	}
	if *gelfTcpPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfTcp(*gelfTcpPort, s.in)
		})
	}
	if *gelfHttpPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfHttp(*gelfHttpPort, s.in)
		})
	}

	log.Printf("loading filters config file %s", filters)
	tree, e := toml.LoadFile(filters)
	if e != nil {
		err = e
		return
	}
	log.Printf("building matchers from %s", tree.Get("filters"))
	if s.Matchers, err = BuildMatchers(tree); err != nil {
		return
	}
	log.Printf("matchers: %v", s.Matchers)

	log.Printf("building destinations from %s", tree.Get("destinations"))
	if s.Alerters, err = BuildAlerters(tree); err != nil {
		return
	}
	log.Printf("alerters: %v", s.Alerters)

	log.Printf("building rules")
	if s.Rules, err = BuildRules(tree, s.Matchers, s.Alerters); err != nil {
		return
	}
	log.Printf("rules: %v", s.Rules)

	return s, nil
}
