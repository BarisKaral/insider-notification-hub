package httpclient

import "time"

type Config struct {
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	DefaultHeaders map[string]string
}
