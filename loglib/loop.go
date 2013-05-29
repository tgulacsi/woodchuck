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
