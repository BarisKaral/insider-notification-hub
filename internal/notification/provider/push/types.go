package push

import "time"

// PushClientConfig holds configuration for the Push provider client.
type PushClientConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}
