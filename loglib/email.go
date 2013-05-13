package loglib

import (
	"net/smtp"
	"strings"
)

type emailSender struct {
	hostport, from string
	auth           smtp.Auth
}

func (es emailSender) Send(to, subject, body string) error {
	return smtp.SendMail(es.hostport, es.auth, es.from,
		[]string{to}, []byte(body))
}

func NewEmailSender(from, hostport, auth string) (es emailSender) {
	es = emailSender{hostport: hostport, from: from}
	if auth != "" {
		host := hostport
		if i := strings.Index(hostport, ":"); i >= 0 {
			host = hostport[:i]
		}
		i := strings.Index(auth, "/")
		username := auth[:i]
		password := auth[i+1:]
		es.auth = smtp.PlainAuth("", username, password, host)
	}
	return es
}
