package main

import (
	"flag"
	"github.com/tgulacsi/woodchuck/loglib"
	"log"
)

var gelfUdpAddr = flag.Int("-g", 12201, "UDP listen port")

func main() {
	if err := loglib.LoadConfig("config.toml", "filters.toml"); err != nil {
		log.Fatalf("error loading config: %s", err)
	}
	in := make(chan *loglib.Message)
	go eventListener(in)
	if err := loglib.ListenGelfUdp(*gelfUdpAddr, in); err != nil {
		log.Fatalf("error listening on %s: %s", *gelfUdpAddr, err)
	}
}

func eventListener(in <-chan *loglib.Message) {
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
	}
}
