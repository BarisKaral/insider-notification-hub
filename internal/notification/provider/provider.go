package provider

import "context"

// NotificationProvider sends notifications through a specific channel.
type NotificationProvider interface {
	Send(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
}
