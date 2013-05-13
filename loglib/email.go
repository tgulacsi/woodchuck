package loglib

import (
	"net/smtp"
	"strings"
)

type emailSender struct {
	hostport, from string
	auth           smtp.Auth
}

func (es emailSender) Send(to []string, subject string, body []byte) error {
	return smtp.SendMail(es.hostport, es.auth, es.from,
		to, body)
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
