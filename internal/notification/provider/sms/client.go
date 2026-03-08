package sms

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/bariskaral/insider-notification-hub/pkg/httpclient"
)

// SMSClient is the HTTP client for the SMS provider.
type SMSClient struct {
	httpClient httpclient.HTTPClient
	baseURL    string
}

// NewSMSClient creates a new SMS provider HTTP client.
func NewSMSClient(config SMSClientConfig) *SMSClient {
	httpClientConfig := httpclient.HTTPClientConfig{
		Timeout:    config.Timeout,
		MaxRetries: config.MaxRetries,
		RetryDelay: 1 * time.Second,
		DefaultHeaders: map[string]string{
			"x-ins-auth-key": config.AuthKey,
		},
	}
	return &SMSClient{
		httpClient: httpclient.NewHTTPClient(httpClientConfig),
		baseURL:    config.URL,
	}
}

// Post sends a traced HTTP POST request to the SMS provider.
func (c *SMSClient) Post(ctx context.Context, body interface{}) (*http.Response, error) {
	ctx, span := otel.Tracer("notification-hub").Start(ctx, "provider.sms.send")
	defer span.End()
	span.SetAttributes(attribute.String("provider.channel", "sms"))

	resp, err := c.httpClient.Post(ctx, c.baseURL, body, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}
