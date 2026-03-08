package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/baris/notification-hub/pkg/httpclient"
)

// ProviderClient sends notifications to the external provider.
type ProviderClient interface {
	Send(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
}

type providerClient struct {
	httpClient httpclient.HTTPClient
	baseURL    string
	authKey    string
}

var _ ProviderClient = (*providerClient)(nil)

// NewProviderClient creates a new provider client with the given configuration.
func NewProviderClient(cfg ProviderConfig) ProviderClient {
	httpCfg := httpclient.HTTPClientConfig{
		Timeout:    cfg.Timeout,
		MaxRetries: cfg.MaxRetries,
		RetryDelay: 1 * time.Second,
		DefaultHeaders: map[string]string{
			"x-ins-auth-key": cfg.AuthKey,
		},
	}

	return &providerClient{
		httpClient: httpclient.NewHTTPClient(httpCfg),
		baseURL:    cfg.URL,
		authKey:    cfg.AuthKey,
	}
}

// Send sends a notification request to the external provider.
func (c *providerClient) Send(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error) {
	ctx, span := otel.Tracer("provider").Start(ctx, "provider.Send")
	defer span.End()

	span.SetAttributes(attribute.String("provider.channel", req.Channel))

	resp, err := c.httpClient.Post(ctx, c.baseURL, req, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "provider connection failed")
		return nil, fmt.Errorf("%w: %v", ErrProviderConnectionFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read response")
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrProviderConnectionFailed, err)
	}

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("%w: status %d, body: %s", ErrProviderRejected, resp.StatusCode, string(body))
		span.RecordError(err)
		span.SetStatus(codes.Error, "provider rejected")
		return nil, err
	}

	var providerResp ProviderResponse
	if err := json.Unmarshal(body, &providerResp); err != nil || providerResp.MessageID == "" {
		// Non-JSON or unexpected response — generate synthetic response
		providerResp = ProviderResponse{
			MessageID: uuid.New().String(),
			Status:    "accepted",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	return &providerResp, nil
}
