package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/bariskaral/insider-notification-hub/internal/notification/provider"
)

// PushProvider sends notifications via the Push channel.
type PushProvider struct {
	client *PushClient
}

var _ provider.NotificationProvider = (*PushProvider)(nil)

// NewPushProvider creates a new Push provider.
func NewPushProvider(config PushClientConfig) *PushProvider {
	return &PushProvider{client: NewPushClient(config)}
}

// Send sends a notification through the Push provider.
func (p *PushProvider) Send(ctx context.Context, request *provider.ProviderRequest) (*provider.ProviderResponse, error) {
	response, err := p.client.Post(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", provider.ErrProviderConnectionFailed, err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", provider.ErrProviderConnectionFailed, err)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status %d, body: %s", provider.ErrProviderRejected, response.StatusCode, string(body))
	}

	var providerResponse provider.ProviderResponse
	if err := json.Unmarshal(body, &providerResponse); err != nil || providerResponse.MessageID == "" {
		providerResponse = provider.ProviderResponse{
			MessageID: uuid.New().String(),
			Status:    "accepted",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	return &providerResponse, nil
}
