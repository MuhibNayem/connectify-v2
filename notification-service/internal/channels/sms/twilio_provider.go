package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// TwilioProvider implements SMSProvider using Twilio
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromPhone  string
	apiURL     string
}

func NewTwilioProvider(accountSID, authToken, fromPhone string) *TwilioProvider {
	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromPhone:  fromPhone,
		apiURL:     "https://api.twilio.com/2010-04-01/Accounts/" + accountSID + "/Messages.json",
	}
}

func (t *TwilioProvider) SendSMS(ctx context.Context, to, body string) error {
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromPhone)
	data.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("twilio API error: status %d", resp.StatusCode)
}

func (t *TwilioProvider) Name() string {
	return "twilio"
}
