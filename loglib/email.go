/*
   Copyright 2013 Tamás Gulácsi

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package loglib

import (
	"net/smtp"
	"strings"
)

type emailSender struct {
	hostport, from string
	auth           smtp.Auth
}

func NewEmailSender(from, hostport, auth string) (es emailSender) {
	host := hostport
	if i := strings.Index(hostport, ":"); i >= 0 {
		host = hostport[:i]
	} else {
		hostport = hostport + ":25"
	}
	es = emailSender{hostport: hostport, from: from}
	if auth != "" {
		i := strings.Index(auth, "/")
		username := auth[:i]
		password := auth[i+1:]
		es.auth = smtp.PlainAuth("", username, password, host)
	}
	return es
}

// sends email to the specified addresses
func (es emailSender) Send(to []string, subject string, body []byte) error {
	return smtp.SendMail(es.hostport, es.auth, es.from,
		to, body)
}
