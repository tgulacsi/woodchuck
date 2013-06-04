// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

package loglib

import (
	"fmt"
	"github.com/sfreiberg/gotwilio"
)

type twilioClient struct {
	client *gotwilio.Twilio
	from   string
}

// NewTwilio returns a new Twilio SMS transport
func NewTwilio(from, sid, token string) twilioClient {
	twilio := gotwilio.NewTwilioClient(*twilioSid, *twilioToken)
	return twilioClient{client: twilio, from: from}
}

// Send sends an SMS
func (tc twilioClient) Send(to, message string) error {
	_, exc, err := tc.client.SendSMS(tc.from, to, message, "", "")
	if err == nil && exc != nil {
		return fmt.Errorf("%s: %s\n%s", exc.Message, exc.Code, exc.MoreInfo)
	}
	return nil
}
