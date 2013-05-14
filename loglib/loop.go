package loglib

import (
	"log"
)

func (s *Server) Start() {
	for i, fun := range s.routines {
		go fun()
		s.routines[i] = nil
	}
	s.routines = nil
}

func (s *Server) Serve() {
	s.Start()

	var rule Rule
	var err error

	for m := range s.in {
		if s.store != nil {
			s.store <- m
		}
		log.Printf("got %#v", m)
		if m.Level <= ERROR {
			log.Printf("ERROR from %s@%s: %s\n%s", m.Facility, m.Host, m.Short, m.Full)
		}
		for _, rule = range s.Rules {
			if rule.Match(m) {
				log.Printf("rule %s matches %s", rule, m)
				if err = rule.Do(m, s); err != nil {
					log.Printf("error doing %s: %s", rule, err)
				}
			}
		}
	}
}
