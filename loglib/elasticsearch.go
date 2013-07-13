// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

package loglib

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ElasticSearchPathPrefix is the path prefix storing data in ElasticSearch
const ElasticSearchPathPrefix = "/woodchuck/gelf"

// StoreCh is the channel for messages to be stored
var StoreCh chan *Message

// ElasticSearch context
type ElasticSearch struct {
	URL    *url.URL
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
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

// NewElasticSearch returns a new ElasticSearch message store
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
	return &ElasticSearch{URL: u, client: http.DefaultClient}
}

// Store stores a message
func (es ElasticSearch) Store(m *Message) (*esResponse, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 512))
	_, _ = io.WriteString(buf, `{"@timestamp": "`)
	_, _ = io.WriteString(buf, time.Unix(m.TimeUnix, 0).Format(time.RFC3339))
	_, _ = io.WriteString(buf, `", "gelf": `)
	var (
		resp *http.Response
		err  error
	)
	if err = json.NewEncoder(buf).Encode(m); err != nil {
		return nil, err
	}
	_, _ = buf.Write([]byte{'}'})
	u := *es.URL
	u.Path += ElasticSearchPathPrefix
	if resp, err = es.client.Post(u.String(), "application/json",
		bytes.NewReader(buf.Bytes())); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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
