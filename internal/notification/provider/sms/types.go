package sms

import "time"

// SMSClientConfig holds configuration for the SMS provider client.
type SMSClientConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}
