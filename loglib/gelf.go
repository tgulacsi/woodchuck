package loglib

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"github.com/tgulacsi/go-gelf/gelf"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
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
	return fmt.Sprintf("%s %s@%s: %s %s", LevelNames[m.Level], m.Facility, m.Host,
		time.Unix(m.TimeUnix, 0).Format(time.RFC3339),
		m.Short)
}
func (m *Message) Long() string {
	return fmt.Sprintf("%s\n%s:%d\n\n%s", m.String(), m.File, m.Line, m.Full)
}

func FromGelfJson(text []byte, m *Message) error {
	return json.Unmarshal(text, m)
}

func ListenGelfUdp(port int, ch chan<- *Message) error {
	log.Printf("start listening on :%d", port)
	r, err := gelf.NewReader(":" + strconv.Itoa(port))
	if err != nil {
		return err
	}
	var (
		gm *gelf.Message
	)
	for {
		if gm, err = r.ReadMessage(); err != nil {
			return fmt.Errorf("error reading message: %s", err)
			log.Fatalf("error reading message: %s", err)
			continue
		}
		ch <- AsMessage(gm)
	}
	log.Printf("stopped listening on :%d", port)
	return nil
}

var (
	magicZlib = []byte{0x78}
	magicGzip = []byte{0x1f, 0x8b}
)

func ListenGelfTcp(port int, ch chan<- *Message) error {
	log.Printf("start listening on :%d", port)
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	var (
		gm *gelf.Message
	)
	handle := func(r io.ReadCloser) {
		defer r.Close()
		if err = UnboxGelf(r, gm); err != nil {
			log.Printf("error unboxing %s: %s", r, err)
			return
		}
		ch <- AsMessage(gm)
	}
	var conn net.Conn
	for {
		if conn, err = ln.Accept(); err != nil {
			log.Printf("error accepting: %s", err)
			continue
		}
		go handle(conn)
	}
	return nil
}

func ListenGelfHttp(port int, ch chan<- *Message) error {
	var (
		gm *gelf.Message
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if r.Method != "POST" {
			w.WriteHeader(400)
			w.Write([]byte("only POST is acceptable"))
			return
		}
		if err := UnboxGelf(r.Body, gm); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("error unboxing gelf message: %s", err)))
			return
		}
		w.WriteHeader(201)
		w.Write([]byte{})
		ch <- AsMessage(gm)
		return
	}
	s := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: http.HandlerFunc(handler)}
	s.ListenAndServe()
	return nil
}

func decompress(r io.Reader) (rc io.ReadCloser, err error) {
	br := bufio.NewReader(r)
	var head []byte
	if head, err = br.Peek(2); err != nil {
		log.Printf("cannot peek into %s: %s", r, err)
		return
	}
	rc, err = ioutil.NopCloser(br), nil
	if bytes.Equal(head[:len(magicGzip)], magicGzip) {
		rc, err = gzip.NewReader(br)
	} else if bytes.Equal(head[:len(magicZlib)], magicZlib) {
		rc, err = zlib.NewReader(br)
	}
	return
}

func UnboxGelf(r io.ReadCloser, m *gelf.Message) (err error) {
	if r, err = decompress(r); err != nil {
		r.Close()
		return
	}
	err = json.NewDecoder(r).Decode(m)
	r.Close()
	return err
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
