package email

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/baris/notification-hub/pkg/httpclient"
)

// EmailClient is the HTTP client for the Email provider.
type EmailClient struct {
	httpClient httpclient.HTTPClient
	baseURL    string
}

// EmailClientConfig holds configuration for the Email provider client.
type EmailClientConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}

// NewEmailClient creates a new Email provider HTTP client.
func NewEmailClient(config EmailClientConfig) *EmailClient {
	httpClientConfig := httpclient.HTTPClientConfig{
		Timeout:    config.Timeout,
		MaxRetries: config.MaxRetries,
		RetryDelay: 1 * time.Second,
		DefaultHeaders: map[string]string{
			"x-ins-auth-key": config.AuthKey,
		},
	}
	return &EmailClient{
		httpClient: httpclient.NewHTTPClient(httpClientConfig),
		baseURL:    config.URL,
	}
}

// Post sends a traced HTTP POST request to the Email provider.
func (c *EmailClient) Post(ctx context.Context, body interface{}) (*http.Response, error) {
	ctx, span := otel.Tracer("notification-hub").Start(ctx, "provider.email.send")
	defer span.End()
	span.SetAttributes(attribute.String("provider.channel", "email"))

	resp, err := c.httpClient.Post(ctx, c.baseURL, body, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}
