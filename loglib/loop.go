// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

package loglib

import (
	"log"
)

// Start starts the needed support goroutines
func (s *Server) Start() {
	for i, fun := range s.routines {
		go fun()
		s.routines[i] = nil
	}
	s.routines = nil
}

// Serve receives messages
func (s *Server) Serve() {
	s.Start()

	var rule Rule
	var err error

	for m := range s.in {
		if s.store != nil {
			s.store <- m
		}
		log.Printf("got %#v", m)
		if LogLevel(m.Level) <= ERROR {
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
