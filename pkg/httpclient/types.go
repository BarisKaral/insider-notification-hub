package httpclient

import "time"

type HTTPClientConfig struct {
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	DefaultHeaders map[string]string
}
