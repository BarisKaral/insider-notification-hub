package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/baris/notification-hub/internal/notification/provider"
)

// PushProvider sends notifications via the Push channel.
type PushProvider struct {
	client *PushClient
}

var _ provider.NotificationProvider = (*PushProvider)(nil)

// NewPushProvider creates a new Push provider.
func NewPushProvider(cfg PushClientConfig) *PushProvider {
	return &PushProvider{client: NewPushClient(cfg)}
}

// Send sends a notification through the Push provider.
func (p *PushProvider) Send(ctx context.Context, req *provider.ProviderRequest) (*provider.ProviderResponse, error) {
	resp, err := p.client.Post(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", provider.ErrProviderConnectionFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", provider.ErrProviderConnectionFailed, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status %d, body: %s", provider.ErrProviderRejected, resp.StatusCode, string(body))
	}

	var providerResp provider.ProviderResponse
	if err := json.Unmarshal(body, &providerResp); err != nil || providerResp.MessageID == "" {
		providerResp = provider.ProviderResponse{
			MessageID: uuid.New().String(),
			Status:    "accepted",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	return &providerResp, nil
}
