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
	"strings"
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
		m.Short, time.Unix(m.TimeUnix, 0).Format(time.RFC3339))
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
		}
		ch <- AsMessage(gm)
	}
	//log.Printf("stopped listening on :%d", port)
	//return nil
}

var (
	magicZlib = []byte{0x78, 0x9c}
	magicGzip = []byte{0x1f, 0x8b}
)

func ListenGelfTcp(port int, ch chan<- *Message) error {
	log.Printf("start listening on :%d", port)
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	handle := func(r io.ReadCloser) {
		gm := &gelf.Message{}
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
	//return nil
}

func ListenGelfHttp(port int, ch chan<- *Message) error {
	var (
		rb  io.ReadCloser
		b   []byte
		i32 int
		e   error
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		gm := &gelf.Message{}
		if r.URL != nil && r.URL.RawQuery != "" {
			q := r.URL.Query()
			gm.Version = q.Get("version")
			gm.Host = q.Get("host")
			gm.Short = q.Get("short_message")
			s := q.Get("timestamp")
			if s != "" {
				i := strings.Index(s, ".")
				if i > 0 {
					s = s[:i]
				}
				if gm.TimeUnix, e = strconv.ParseInt(s, 10, 64); e != nil {
					log.Printf("error parsing timestamp %s", s)
				}
			}
			if i32, e = strconv.Atoi(q.Get("level")); e != nil {
				log.Printf("error parsing level %s", q.Get("level"))
			} else {
				gm.Level = int32(i32)
			}
			gm.Facility = q.Get("facility")
			gm.File = q.Get("file")
			if gm.Line, e = strconv.Atoi(q.Get("line")); e != nil {
				log.Printf("error parsing line %s", q.Get("line"))
			}
			for k, v := range q {
				if k[0] == '_' {
					if gm.Extra == nil {
						gm.Extra = make(map[string]interface{}, len(q))
					}
					gm.Extra[k] = v
				}
			}
			if r.Method == "POST" {
				if rb, e = decompress(r.Body); e != nil {
					log.Printf("error decompressing body: %s", e)
				}
				b, e = ioutil.ReadAll(rb)
				rb.Close()
				r.Body.Close()
				if e != nil {
					log.Printf("error reading body: %s", e)
				}
				gm.Full = string(b)
			}
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
	} else {
		log.Printf("WARN not compressed? %x", head)
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
