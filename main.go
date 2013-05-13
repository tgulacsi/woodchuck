package main

import (
	"github.com/tgulacsi/woodchuck/loglib"
	"log"
)

func main() {
	if err := loglib.LoadConfig("config.toml", "filters.toml"); err != nil {
		log.Fatalf("error loading config: %s", err)
	}
	in := make(chan *loglib.Message)
	go eventListener(in)
	if err := loglib.ListenGelfUdp(in); err != nil {
		log.Fatalf("error listening: %s", err)
	}
}

func eventListener(in <-chan *loglib.Message) {
	var rule loglib.Rule
	var err error
	for m := range in {
		if m.Extra != nil && m.Full == "" {
			if f, ok := m.Extra["_full_message"]; ok {
				if fs, ok := f.(string); ok {
					if !(fs == "''" || fs == `""`) {
						m.Full = fs
						delete(m.Extra, "_full_message")
					}
				}
			}
		}
		if loglib.StoreCh != nil {
			loglib.StoreCh <- m
		}
		//log.Printf("got %#v", m)
		if m.Level <= loglib.ERROR {
			log.Printf("ERROR from %s@%s: %s\n%s", m.Facility, m.Host, m.Short, m.Full)
		}
		for _, rule = range loglib.Rules {
			if rule.Match(m) {
				log.Printf("rule %s matches %s", rule, m)
				if err = rule.Do(m); err != nil {
					log.Printf("error doing %s: %s", rule, err)
				}
			}
		}
	}
}
