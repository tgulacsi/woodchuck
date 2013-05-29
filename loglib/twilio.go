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
