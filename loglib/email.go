package loglib

import (
	"net/smtp"
	"strings"
)

func SendEmail(from string, to []string, subject, body string) error {
	var auth smtp.Auth
	if *smtpAuth != "" {
		host := *smtpHostport
		if i := strings.Index(*smtpHostport, ":"); i >= 0 {
			host = (*smtpHostport)[:i]
		}
		i := strings.Index(*smtpAuth, "/")
		username := (*smtpAuth)[:i]
		password := (*smtpAuth)[i+1:]
		auth = smtp.PlainAuth("", username, password, host)
	}
	return smtp.SendMail(*smtpHostport, auth, from,
		to, []byte(body))
}
