package loglib

import (
	"bytes"
	"github.com/tgulacsi/go-xmlrpc"
	"log"
    "io"
	"net/http"
	"net/url"
)

type mantisSender struct {
	xmlrpcPhp string
}

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

func (ms mantisSender) Send(uri, subject, body string) (int, error) {
	mantisUrl, projectName, category, username, password, err := splitUrl(uri)
	if err != nil {
		return -1, err
	}
	if ms.xmlrpcPhp != "" {
		mantisUrl += "/" + ms.xmlrpcPhp
	}
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

func NewMantisSender(xmlrpcPhp string) (ms mantisSender) {
	return mantisSender{xmlrpcPhp}
}

func Call(uri, username, password, name string, args ...interface{}) (interface{}, *xmlrpc.Fault, error) {
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
	defer r.Body.Close()

	buf.Reset()
	_, v, f, e := xmlrpc.Unmarshal(io.TeeReader(r.Body, buf))
	log.Printf("got\n%s", buf.Bytes())
	return v, f, e
}
