package push

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/baris/notification-hub/pkg/httpclient"
)

// PushClient is the HTTP client for the Push provider.
type PushClient struct {
	httpClient httpclient.HTTPClient
	baseURL    string
}

// PushClientConfig holds configuration for the Push provider client.
type PushClientConfig struct {
	URL        string
	AuthKey    string
	Timeout    time.Duration
	MaxRetries int
}

// NewPushClient creates a new Push provider HTTP client.
func NewPushClient(cfg PushClientConfig) *PushClient {
	httpCfg := httpclient.HTTPClientConfig{
		Timeout:    cfg.Timeout,
		MaxRetries: cfg.MaxRetries,
		RetryDelay: 1 * time.Second,
		DefaultHeaders: map[string]string{
			"x-ins-auth-key": cfg.AuthKey,
		},
	}
	return &PushClient{
		httpClient: httpclient.NewHTTPClient(httpCfg),
		baseURL:    cfg.URL,
	}
}

// Post sends a traced HTTP POST request to the Push provider.
func (c *PushClient) Post(ctx context.Context, body interface{}) (*http.Response, error) {
	ctx, span := otel.Tracer("notification-hub").Start(ctx, "provider.push.send")
	defer span.End()
	span.SetAttributes(attribute.String("provider.channel", "push"))

	resp, err := c.httpClient.Post(ctx, c.baseURL, body, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}
