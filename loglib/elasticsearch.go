package loglib

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const ElasticSearchPathPrefix = "/woodchuck/gelf"

var StoreCh chan *Message

type ElasticSearch struct {
	Url    *url.URL
	client *http.Client
}

// {
//    "ok" : true,
//    "_index" : "twitter",
//    "_type" : "tweet",
//    "_id" : "6a8ca01c-7896-48e9-81cc-9f70661fcb32",
//    "_version" : 1
//}
type esResponse struct {
	Ok      bool   `json:"ok"`
	Index   string `json:"_index"`
	Type    string `json:"_type"`
	Id      string `json:"_id"`
	Version int    `json:"_version"`
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

func NewElasticSearch(urls string, ttld int) *ElasticSearch {
	u, err := url.Parse(urls)
	if err != nil {
		log.Fatalf("bad url %s: %s", urls, err)
		return nil
	}
	if ttld > 0 {
		q := u.Query()
		q.Set("ttl", strconv.Itoa(ttld)+"d")
		u.RawQuery = q.Encode()
	}
	return &ElasticSearch{Url: u, client: http.DefaultClient}
}

func (es ElasticSearch) Store(m *Message) (*esResponse, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 512))
	_, _ = buf.Write([]byte(`{"gelf": `))
	var (
		resp *http.Response
		err  error
	)
	if err = json.NewEncoder(buf).Encode(m); err != nil {
		return nil, err
	}
	_, _ = buf.Write([]byte{'}'})
	u := *es.Url
	u.Path += ElasticSearchPathPrefix
	if resp, err = es.client.Post(u.String(), "application/json",
		bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}
	esR := new(esResponse)
	if err = json.NewDecoder(resp.Body).Decode(esR); err != nil {
		return esR, err
	}
	return esR, err
}

func storeEs(url string, ttld int, in <-chan *Message) {
	es := NewElasticSearch(url, ttld)
	var (
		resp *esResponse
		err  error
	)
	for m := range in {
		if resp, err = es.Store(m); err != nil {
			log.Printf("error storing message: %s", err)
			continue
		}
		log.Printf("message stored as %#v", resp)
	}
}
