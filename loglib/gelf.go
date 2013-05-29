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
	"encoding/json"
	"fmt"
	"github.com/SocialCodeInc/go-gelf/gelf"
	"time"
)

const (
	EMERGENCY = iota
	ALERT
	CRITICAL
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
)

var LevelNames = [8]string{"EMERGENCY", "ALERT", "CRITICAL", "ERROR",
	"WARNING", "NOTICE", "INFO", "DEBUG"}

//var defaultGelf = gelf.New(gelf.Config{})

//type Message struct {
//    Version  string                 `json:"version"`
//    Host     string                 `json:"host"`
//    Short    string                 `json:"short_message"`
//    Full     string                 `json:"full_message"`
//    TimeUnix int64                  `json:"timestamp"`
//    Level    int32                  `json:"level"`
//    Facility string                 `json:"facility"`
//    File     string                 `json:"file"`
//    Line     int                    `json:"line"`
//    Extra    map[string]interface{} `json:"-"`
//}
type Message gelf.Message

func (m *Message) MarshalJSON() ([]byte, error) {
	return ((*gelf.Message)(m)).MarshalJSON()
}
func (m *Message) UnmarshalJSON(data []byte) error {
	return ((*gelf.Message)(m)).UnmarshalJSON(data)
}
func (m *Message) String() string {
	return fmt.Sprintf("%s %s@%s: %s", LevelNames[m.Level], m.Facility, m.Host,
		m.Short)
}
func (m *Message) Long() string {
	return fmt.Sprintf("%s\n%s\n%s:%d\n\n%s", m.String(),
		time.Unix(m.TimeUnix, 0).Format(time.RFC3339), m.File, m.Line, m.Full)
}

func FromGelfJson(text []byte, m *Message) error {
	return json.Unmarshal(text, m)
}

func AsMessage(gm *gelf.Message) *Message {
	m := (*Message)(gm)
	m.Fix()
	return m
}

func (m *Message) Fix() {
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
}
