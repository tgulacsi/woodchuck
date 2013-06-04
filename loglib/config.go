// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

// Package loglib implements some remote logging mechanisms
package loglib

import (
	"github.com/pelletier/go-toml"
	"github.com/stvp/go-toml-config"
	"log"
	"time"
)

var (
	//TransportConfig is the configset for transportation settings
	TransportConfig = config.NewConfigSet("transportation settings", config.ExitOnError)
	from            = TransportConfig.String("from", "woodchuck")
	gelfUdpPort     = TransportConfig.Int("gelf.udp", 12201)
	gelfTcpPort     = TransportConfig.Int("gelf.tcp", 0)
	gelfHTTPPort    = TransportConfig.Int("gelf.http", 0)

	twilioSid   = TransportConfig.String("twilio.sid", "")
	twilioToken = TransportConfig.String("twilio.token", "")
	twilioRate  = TransportConfig.Int("twilio.rate", 1800)

	smtpHostport = TransportConfig.String("smtp.hostport", ":25")
	smtpAuth     = TransportConfig.String("smtp.auth", "")
	smtpRate     = TransportConfig.Int("smtp.rate", 600)

	mantisXmlrpc = TransportConfig.String("mantis.xmlrpc", "xmlrpc_vv.php")
	mantisRate   = TransportConfig.Int("mantis.rate", 3600)

	esURL = TransportConfig.String("elasticsearch.url", "http://localhost:9200")
	esTTL = TransportConfig.Int("elasticsearch.ttl", 90)
)

// SMSSender is the SMS sender interface (just to and from)
type SMSSender interface {
	Send(to, message string) error
}

// EmailSender is the email sender interface (multiple to, a subject and a []byte body)
type EmailSender interface {
	Send(to []string, subject string, body []byte) error
}

// MantisSender is an interface for an issue tracker injector
type MantisSender interface {
	Send(uri, subject, body string) (int, error)
}

// SenderProvider is an interface for returning the specific senders
type SenderProvider interface {
	GetSMSSender(string) SMSSender
	GetEmailSender(string) EmailSender
	GetMantisSender(string) MantisSender
}

// Server is the server context
type Server struct {
	in, store chan *Message
	sms       SMSSender
	email     EmailSender
	mantis    MantisSender
	Rules     []Rule
	Matchers  map[string]Matcher
	Alerters  map[string]Alerter
	routines  []func()
	rates     struct {
		limiter            RateLimiter
		sms, email, mantis time.Duration
	}
}

// GetSMSSender returns the SMSSender, implementing rate limiting
func (s Server) GetSMSSender(txt string) SMSSender {
	if s.rates.limiter != nil && s.rates.sms > 0 && !s.rates.limiter.Put(s.rates.sms, txt) {
		return nil
	}
	return s.sms
}

// GetEmailSender returns the EmailSender, if not above rate limit
func (s Server) GetEmailSender(txt string) EmailSender {
	if s.rates.limiter != nil && s.rates.email > 0 && !s.rates.limiter.Put(s.rates.email, txt) {
		return nil
	}
	return s.email
}

// GetMantisSender returns the MantisSender, if not above rate limit
func (s Server) GetMantisSender(txt string) MantisSender {
	if s.rates.limiter != nil && s.rates.mantis > 0 && !s.rates.limiter.Put(s.rates.mantis, txt) {
		return nil
	}
	return s.mantis
}

// LoadConfig loads the config read from the transports and filters TOML files
func LoadConfig(transports, filters string) (s *Server, err error) {
	log.Printf("loading transports config file %s", filters)
	if err = TransportConfig.Parse(transports); err != nil {
		return
	}
	s = &Server{routines: make([]func(), 0, 4), in: make(chan *Message)}
	s.rates.limiter = NewRateLimiter(time.Hour)
	if *esURL != "" {
		log.Printf("starting storage goroutine for %s", *esURL)
		s.store = make(chan *Message)
		s.routines = append(s.routines, func() {
			storeEs(*esURL, *esTTL, s.store)
		})
	}
	if *twilioSid != "" {
		s.sms = NewTwilio("+1 858-500-3858", *twilioSid, *twilioToken)
		s.rates.sms = time.Duration(*twilioRate) * time.Second
	}
	if *smtpHostport != "" {
		s.email = NewEmailSender(*from, *smtpHostport, *smtpAuth)
		s.rates.email = time.Duration(*smtpRate) * time.Second
	}
	s.mantis = NewMantisSender()
	s.rates.mantis = time.Duration(*mantisRate) * time.Second
	if *gelfUdpPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfUDP(*gelfUdpPort, s.in)
		})
	}
	if *gelfTcpPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfTCP(*gelfTcpPort, s.in)
		})
	}
	if *gelfHTTPPort > 0 {
		s.routines = append(s.routines, func() {
			ListenGelfHTTP(*gelfHTTPPort, s.in)
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
