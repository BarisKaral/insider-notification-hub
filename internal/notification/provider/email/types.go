package email

import "time"

// EmailClientConfig holds configuration for the Email provider client.
type EmailClientConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}
