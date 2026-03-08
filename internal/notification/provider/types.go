package provider

import "time"

// ProviderRequest is the payload sent to the external notification provider.
type ProviderRequest struct {
	To      string `json:"to"`
	Channel string `json:"channel"`
	Content string `json:"content"`
}

// ProviderResponse is the response from the external notification provider.
type ProviderResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ProviderConfig holds shared configuration for all providers.
type ProviderConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}
