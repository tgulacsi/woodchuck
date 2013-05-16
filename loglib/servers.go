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
)

// listen on the given UDP port for possibly chunked GELF messages
// put every complete message into the channel
func ListenGelfUdp(port int, ch chan<- *Message) error {
	log.Printf("start listening on :%d", port)
	r, err := gelf.NewReader(":" + strconv.Itoa(port))
	if err != nil {
		return err
	}
	var gm *gelf.Message
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

// listen on the given TCP port for full, possibly compressed GELF messages
// put every message into the channel
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

// listhen on the given HTTP port for multipart/form POST requests such as
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

// parse values from url.Values into the gelf Message
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

// decompress with zlib or gzip (or nothing) as the magic first two bytes says
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

// unbox gelf message: decompress and decode from JSON
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
