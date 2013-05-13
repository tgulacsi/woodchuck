package loglib

import (
	"encoding/json"
	"fmt"
	"github.com/tgulacsi/go-gelf/gelf"
	"log"
	"strconv"
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

//var defaultGelf = gelf.New(gelf.Config{})

type Message gelf.Message

func (m *Message) MarshalJSON() ([]byte, error) {
	return ((*gelf.Message)(m)).MarshalJSON()
}
func (m *Message) UnmarshalJSON(data []byte) error {
	return ((*gelf.Message)(m)).UnmarshalJSON(data)
}

func FromGelfJson(text []byte, m *Message) error {
	return json.Unmarshal(text, m)
}

func ListenGelfUdp(ch chan<- *Message) error {
	port := *gelfUdpPort
	log.Printf("start listening on :%d", port)
	r, err := gelf.NewReader(":" + strconv.Itoa(port))
	if err != nil {
		return err
	}
	var (
		m *gelf.Message
	)
	for {
		if m, err = r.ReadMessage(); err != nil {
			return fmt.Errorf("error reading message: %s", err)
			log.Fatalf("error reading message: %s", err)
			continue
		}
		ch <- (*Message)(m)
	}
	log.Printf("stopped listening on :%d", port)
	return nil
}
