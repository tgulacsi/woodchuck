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
	"net/url"
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
	log.Fatalf("stopped listening UDP on %d", port)
	return nil
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
	log.Fatalf("End listening TCP on %d", port)
	return nil
}

// curl -v -F timestamp=$(date '+%s') -F short=abraka -F host=$(hostname) -F full=dabra -F facility=proba -F level=6 http://unowebprd:12203/
func ListenGelfHttp(port int, ch chan<- *Message) error {
	var (
		rb io.ReadCloser
		b  []byte
		e  error
		ok bool
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		var gm *gelf.Message
		ok = false
		if r.Body != nil {
			defer r.Body.Close()
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		parsErr := func(err error) {
			ok = false
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			w.Write([]byte{'\n'})
		}
		if r.URL != nil && r.URL.RawQuery != "" {
			gm = &gelf.Message{}
			if e = parseValues(r.URL.Query(), gm); e != nil {
				parsErr(e)
			}
			if r.Method == "POST" {
				if rb, e = decompress(r.Body); e != nil {
					parsErr(fmt.Errorf("error decompressing body: %s", e))
				} else {
					b, e = ioutil.ReadAll(rb)
					rb.Close()
					if e != nil {
						parsErr(fmt.Errorf("error reading body: %s", e))
					} else {
						gm.Full = string(b)
						ok = true
					}
				}
			}
		} else {
			if r.Method != "POST" && r.Method != "PUT" {
				parsErr(fmt.Errorf("POST needed!"))
			} else {
				gm = &gelf.Message{}
				rb = nil
				s := r.FormValue("full")
				if s != "" {
					if rb, e = decompress(bytes.NewReader([]byte(s))); e != nil {
						parsErr(fmt.Errorf("error decompressing full: %s", e))
					}
				} else {
					if mpf, _, e := r.FormFile("full"); e != nil {
						parsErr(e)
					} else {
						if rb, e = decompress(mpf); e != nil {
							parsErr(fmt.Errorf("error decompressing full file: %s", e))
						}
					}
				}
				if rb != nil && e == nil {
					b, e = ioutil.ReadAll(rb)
					rb.Close()
					if e != nil {
						parsErr(fmt.Errorf("error reading full: %s", e))
					} else {
						gm.Full = string(b)
					}
				}
				if e = parseValues(r.Form, gm); e != nil {
					parsErr(e)
				}
				ok = true
			}
		}
		if ok {
			w.WriteHeader(201)
			w.Write([]byte{})
			if gm != nil && gm.Facility != "" {
				ch <- AsMessage(gm)
			}
		}
		return
	}
	s := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: http.HandlerFunc(handler)}
	s.ListenAndServe()
	log.Fatalf("end listening HTTP on port %d", port)
	return nil
}

func parseValues(q url.Values, gm *gelf.Message) (err error) {
	gm.Version = q.Get("version")
	gm.Host = q.Get("host")
	gm.Short = q.Get("short")
	s := q.Get("timestamp")
	if s != "" {
		i := strings.Index(s, ".")
		if i > 0 {
			s = s[:i]
		}
		if gm.TimeUnix, err = strconv.ParseInt(s, 10, 64); err != nil {
			return fmt.Errorf("error parsing timestamp %s: %s", s, err)
		}
	}
	var i32 int
	s = q.Get("level")
	if s != "" {
		if i32, err = strconv.Atoi(s); err != nil {
			return fmt.Errorf("error parsing level %s: %s", q.Get("level"), err)
		} else {
			gm.Level = int32(i32)
		}
	}
	gm.Facility = q.Get("facility")
	gm.File = q.Get("file")
	s = q.Get("line")
	if s != "" {
		if gm.Line, err = strconv.Atoi(s); err != nil {
			return fmt.Errorf("error parsing line %s: %s", q.Get("line"), err)
		}
	}
	for k, v := range q {
		if k[0] == '_' {
			if gm.Extra == nil {
				gm.Extra = make(map[string]interface{}, len(q))
			}
			gm.Extra[k] = v
		}
	}
	return nil
}

func decompress(r io.Reader) (rc io.ReadCloser, err error) {
	br := bufio.NewReader(r)
	var head []byte
	rc, err = ioutil.NopCloser(br), nil
	if head, err = br.Peek(2); err != nil {
		if err == io.EOF {
			return ioutil.NopCloser(bytes.NewReader(nil)), nil
		}
		err = fmt.Errorf("cannot peek into %s: %s", r, err)
		return
	}
	if bytes.Equal(head[:len(magicGzip)], magicGzip) {
		rc, err = gzip.NewReader(br)
	} else if bytes.Equal(head[:len(magicZlib)], magicZlib) {
		rc, err = zlib.NewReader(br)
	} else {
		log.Printf("WARN not compressed? %x", head)
	}
	return
}

func UnboxGelf(rc io.ReadCloser, m *gelf.Message) (err error) {
	var r io.ReadCloser
	if r, err = decompress(rc); err != nil {
		rc.Close()
		return
	}
	err = json.NewDecoder(r).Decode(m)
	r.Close()
	rc.Close()
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
