/*
   Copyright 2013 Tamás Gulácsi

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package loglib

import (
	"github.com/pelletier/go-toml"
	"github.com/stvp/go-toml-config"
	"log"
	"time"
)

var (
	TransportConfig = config.NewConfigSet("transportation settings", config.ExitOnError)
	from            = TransportConfig.String("from", "woodchuck")
	gelfUdpPort     = TransportConfig.Int("gelf.udp", 12201)
	gelfTcpPort     = TransportConfig.Int("gelf.tcp", 0)
	gelfHttpPort    = TransportConfig.Int("gelf.http", 0)

	twilioSid   = TransportConfig.String("twilio.sid", "")
	twilioToken = TransportConfig.String("twilio.token", "")
	twilioRate  = TransportConfig.Int("twilio.rate", 1800)

	smtpHostport = TransportConfig.String("smtp.hostport", ":25")
	smtpAuth     = TransportConfig.String("smtp.auth", "")
	smtpRate     = TransportConfig.Int("smtp.rate", 600)

	mantisXmlrpc = TransportConfig.String("mantis.xmlrpc", "xmlrpc_vv.php")
	mantisRate   = TransportConfig.Int("mantis.rate", 3600)

	esUrl = TransportConfig.String("elasticsearch.url", "http://localhost:9200")
	esTTL = TransportConfig.Int("elasticsearch.ttl", 90)
)

type SMSSender interface {
	Send(to, message string) error
}
type EmailSender interface {
	Send(to []string, subject string, body []byte) error
}
type MantisSender interface {
	Send(uri, subject, body string) (int, error)
}
type SenderProvider interface {
	GetSMSSender(string) SMSSender
	GetEmailSender(string) EmailSender
	GetMantisSender(string) MantisSender
}

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

func (s Server) GetSMSSender(txt string) SMSSender {
	if s.rates.limiter != nil && s.rates.sms > 0 && !s.rates.limiter.Put(s.rates.sms, txt) {
		return nil
	}
	return s.sms
}
func (s Server) GetEmailSender(txt string) EmailSender {
	if s.rates.limiter != nil && s.rates.email > 0 && !s.rates.limiter.Put(s.rates.email, txt) {
		return nil
	}
	return s.email
}
func (s Server) GetMantisSender(txt string) MantisSender {
	if s.rates.limiter != nil && s.rates.mantis > 0 && !s.rates.limiter.Put(s.rates.mantis, txt) {
		return nil
	}
	return s.mantis
}

func LoadConfig(transports, filters string) (s *Server, err error) {
	log.Printf("loading transports config file %s", filters)
	if err = TransportConfig.Parse(transports); err != nil {
		return
	}
	s = &Server{routines: make([]func(), 0, 4), in: make(chan *Message)}
	s.rates.limiter = NewRateLimiter(time.Hour)
	if *esUrl != "" {
		log.Printf("starting storage goroutine for %s", *esUrl)
		s.store = make(chan *Message)
		s.routines = append(s.routines, func() {
			storeEs(*esUrl, *esTTL, s.store)
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
