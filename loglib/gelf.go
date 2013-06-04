// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

package loglib

import (
	"encoding/json"
	"fmt"
	"github.com/SocialCodeInc/go-gelf/gelf"
	"time"
)

// LogLevel is the logging level, copied from syslog
type LogLevel int

const (
	// EMERGENCY is the highest level
	EMERGENCY = LogLevel(iota)
	// ALERT is the alert level
	ALERT
	// CRITICAL syslog level
	CRITICAL
	// ERROR syslog level
	ERROR
	// WARNING syslog level
	WARNING
	// NOTICE syslog level
	NOTICE
	// INFO syslog level
	INFO
	// DEBUG syslog level
	DEBUG
)

// LevelNames is the names of the levels
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

// Message is a gelf.Message wrapper
type Message gelf.Message

// MarshalJSON returns the message marshaled to JSON
func (m *Message) MarshalJSON() ([]byte, error) {
	return ((*gelf.Message)(m)).MarshalJSON()
}

// UnmarshalJSON unmarshals JSON into the message
func (m *Message) UnmarshalJSON(data []byte) error {
	return ((*gelf.Message)(m)).UnmarshalJSON(data)
}

// String returns a short representation of the message
func (m *Message) String() string {
	return fmt.Sprintf("%s %s@%s: %s", LevelNames[m.Level], m.Facility, m.Host,
		m.Short)
}

// Long returns a short representation of the message
func (m *Message) Long() string {
	return fmt.Sprintf("%s\n%s\n%s:%d\n\n%s", m.String(),
		time.Unix(m.TimeUnix, 0).Format(time.RFC3339), m.File, m.Line, m.Full)
}

// FromGelfJSON reads the GELF JSON into the message
func FromGelfJSON(text []byte, m *Message) error {
	return json.Unmarshal(text, m)
}

// AsMessage is a type conversion + some fixes
func AsMessage(gm *gelf.Message) *Message {
	m := (*Message)(gm)
	m.Fix()
	return m
}

// Fix fixes some GELF quirks (_full_message => Full)
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
