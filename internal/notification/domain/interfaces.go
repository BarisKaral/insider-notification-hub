package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// NotificationRepository defines the data access interface for notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	CreateBatch(ctx context.Context, notifications []*Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*Notification, error)
	List(ctx context.Context, filter NotificationListFilter) ([]*Notification, int64, error)
	Update(ctx context.Context, n *Notification) error
	GetByIdempotencyKey(ctx context.Context, key string) (*Notification, error)
	GetForProcessing(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*Notification, error)
	GetDueScheduledNotifications(ctx context.Context) ([]*Notification, error)
}

// NotificationService defines the business logic interface for notifications.
type NotificationService interface {
	Create(ctx context.Context, req NotificationCreateRequest, idempotencyKey *string) (*Notification, error)
	CreateBatch(ctx context.Context, req NotificationBatchCreateRequest) ([]*Notification, uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*Notification, error)
	List(ctx context.Context, filter NotificationListFilter) ([]*Notification, int64, error)
	Cancel(ctx context.Context, id uuid.UUID) (*Notification, error)
	MarkAsProcessing(ctx context.Context, id uuid.UUID) (*Notification, error)
	MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error
	MarkAsQueued(ctx context.Context, id uuid.UUID) error
	MarkAsRetrying(ctx context.Context, id uuid.UUID) error
	RecoverStuckNotifications(ctx context.Context) error
	PublishDueScheduled(ctx context.Context) error
}

// NotificationProducer publishes notifications to message queues.
type NotificationProducer interface {
	Publish(ctx context.Context, n *Notification) error
	PublishBatch(ctx context.Context, notifications []*Notification) error
}

// StatusBroadcaster broadcasts notification status changes (implemented by WebSocket hub).
type StatusBroadcaster interface {
	Broadcast(notificationID string, batchID *string, status string)
}
