package loglib

import (
	"fmt"
	"github.com/sfreiberg/gotwilio"
)

func TwilioSendSMS(from, to, message string) error {
	twilio := gotwilio.NewTwilioClient(*twilioSid, *twilioToken)
	_, exc, err := twilio.SendSMS(from, to, message, "", "")
	if err == nil && exc != nil {
		return fmt.Errorf("%s: %s\n%s", exc.Message, exc.Code, exc.MoreInfo)
	}
	return nil
}
