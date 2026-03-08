package email

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/baris/notification-hub/internal/notification/provider"
)

// EmailProvider sends notifications via the Email channel.
type EmailProvider struct {
	client *EmailClient
}

var _ provider.NotificationProvider = (*EmailProvider)(nil)

// NewEmailProvider creates a new Email provider.
func NewEmailProvider(cfg EmailClientConfig) *EmailProvider {
	return &EmailProvider{client: NewEmailClient(cfg)}
}

// Send sends a notification through the Email provider.
func (p *EmailProvider) Send(ctx context.Context, req *provider.ProviderRequest) (*provider.ProviderResponse, error) {
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
