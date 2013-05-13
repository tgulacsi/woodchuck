package loglib

import (
	"fmt"
	"github.com/sfreiberg/gotwilio"
)

type twilioClient struct {
	client *gotwilio.Twilio
	from   string
}

func NewTwilio(from, sid, token string) twilioClient {
	twilio := gotwilio.NewTwilioClient(*twilioSid, *twilioToken)
	return twilioClient{client: twilio, from: from}
}

func (tc twilioClient) Send(to, message string) error {
	_, exc, err := tc.client.SendSMS(tc.from, to, message, "", "")
	if err == nil && exc != nil {
		return fmt.Errorf("%s: %s\n%s", exc.Message, exc.Code, exc.MoreInfo)
	}
	return nil
}
