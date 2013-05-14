package loglib

import (
	"bytes"
	"github.com/tgulacsi/go-xmlrpc"
	"io"
	"log"
	"net/http"
	"net/url"
)

type callFunc func(subject, body string) (int, error)

type mantisSender struct {
	callers map[string]callFunc
}

// splits url, gets username/password and project and category
func splitUrl(uri string) (mantisUrl, projectName, category, username, password string, err error) {
	u, e := url.Parse(uri)
	if e != nil {
		err = e
		return
	}
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		u.User = nil
	}
	if u.RawQuery != "" {
		projectName = u.Query().Get("project")
		category = u.Query().Get("category")
		u.RawQuery = ""
	}
	mantisUrl = u.String()
	return
}

// creates a new Mantis issue on the given uri
func (ms *mantisSender) Send(uri, subject, body string) (int, error) {
	if ms.callers == nil {
		ms.callers = make(map[string]callFunc, 1)
	}
	call, ok := ms.callers[uri]
	if !ok {
		mantisUrl, projectName, category, username, password, err := splitUrl(uri)
		if err != nil {
			return -1, err
		}
		call = func(subject, body string) (int, error) {
			args := map[string]string{"project_name": projectName, "summary": subject,
				"description": body, "category": category}
			log.Printf("calling %s new_issue(%v)", mantisUrl, args)
			resp, fault, err := Call(mantisUrl, username, password, "new_issue", args)
			log.Printf("got %v, %v, %s", resp, fault, err)
			if err == nil {
				log.Printf("response: %v", resp)
				return -1, fault
			}
			return 0, err
		}
	}
	return call(subject, body)
}

// returns a new Mantis sender
func NewMantisSender() (ms *mantisSender) {
	return &mantisSender{callers: make(map[string]callFunc, 4)}
}

// xmlrpc.Call, but without gzip and Basic Auth and strips non-xml
func Call(uri, username, password, name string, args ...interface{}) (
	interface{}, *xmlrpc.Fault, error) {

	buf := bytes.NewBuffer(nil)
	e := xmlrpc.Marshal(buf, name, args...)
	if e != nil {
		return nil, nil, e
	}
	tr := &http.Transport{
		//ClientConfig:    &tls.Config{RootCAs: pool},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	req, e := http.NewRequest("POST", uri, buf)
	if e != nil {
		return nil, nil, e
	}
	req.SetBasicAuth(username, password)
	r, e := client.Do(req)
	//r, e := client.Post(uri, "text/xml", buf)
	if e != nil {
		return nil, nil, e
	}

	buf.Reset()
	n, e := io.Copy(buf, r.Body)
	r.Body.Close()
	if e != nil {
		return nil, nil, e
	}
	b := buf.Bytes()[:n]
	log.Printf("got\n%s", b)
	i := bytes.Index(b, []byte("<?"))
	if i < 0 {
		return nil, nil, io.EOF
	}
	_, v, f, e := xmlrpc.Unmarshal(bytes.NewReader(b[i:]))
	return v, f, e
}
