package service

import (
	"context"
	"fmt"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/provider"
	"github.com/baris/notification-hub/pkg/logger"
	"github.com/google/uuid"
)

// NotificationProcessingService orchestrates notification workflows across
// the domain service, message producer, and notification providers.
type NotificationProcessingService interface {
	Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error)
	CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error)
	ProcessAndSend(ctx context.Context, id uuid.UUID) (*domain.Notification, bool, error)
	HandleDeliveryFailure(ctx context.Context, id uuid.UUID, retryCount int, maxRetries int) (*domain.Notification, error)
	RecoverStuckNotifications(ctx context.Context) error
	PublishDueScheduled(ctx context.Context) error
}

type notificationProcessingService struct {
	service   NotificationService
	producer  NotificationProducer
	providers map[domain.NotificationChannel]provider.NotificationProvider
}

var _ NotificationProcessingService = (*notificationProcessingService)(nil)

// NewNotificationProcessingService creates a new NotificationProcessingService.
func NewNotificationProcessingService(
	svc NotificationService,
	producer NotificationProducer,
	providers map[domain.NotificationChannel]provider.NotificationProvider,
) NotificationProcessingService {
	return &notificationProcessingService{
		service:   svc,
		producer:  producer,
		providers: providers,
	}
}

func (s *notificationProcessingService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	n, err := s.service.Create(ctx, req, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if n.Status != domain.NotificationStatusScheduled {
		if err := s.producer.Publish(ctx, n); err != nil {
			logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to publish notification")
			return n, nil
		}

		if err := s.service.MarkAsQueued(ctx, n.ID); err != nil {
			logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to mark notification as queued")
			return n, nil
		}

		n.Status = domain.NotificationStatusQueued
	}

	return n, nil
}

func (s *notificationProcessingService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	notifications, batchID, err := s.service.CreateBatch(ctx, req)
	if err != nil {
		return nil, uuid.Nil, err
	}

	var toPublish []*domain.Notification
	for _, n := range notifications {
		if n.Status != domain.NotificationStatusScheduled {
			toPublish = append(toPublish, n)
		}
	}

	if len(toPublish) > 0 {
		if err := s.producer.PublishBatch(ctx, toPublish); err != nil {
			logger.Error().Err(err).Str("batchID", batchID.String()).Msg("failed to publish batch notifications")
			return notifications, batchID, nil
		}

		for _, n := range toPublish {
			if err := s.service.MarkAsQueued(ctx, n.ID); err != nil {
				logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to mark notification as queued")
			} else {
				n.Status = domain.NotificationStatusQueued
			}
		}
	}

	return notifications, batchID, nil
}

func (s *notificationProcessingService) ProcessAndSend(ctx context.Context, id uuid.UUID) (*domain.Notification, bool, error) {
	n, err := s.service.MarkAsProcessing(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if n.Status == domain.NotificationStatusCancelled || n.Status == domain.NotificationStatusSent {
		return n, false, nil
	}

	p, ok := s.providers[n.Channel]
	if !ok {
		return nil, false, fmt.Errorf("no provider for channel %s", n.Channel)
	}

	resp, err := p.Send(ctx, &provider.ProviderRequest{
		To:      n.Recipient,
		Channel: string(n.Channel),
		Content: n.Content,
	})
	if err != nil {
		return nil, false, err
	}

	if err := s.service.MarkAsSent(ctx, id, resp.MessageID); err != nil {
		return nil, false, err
	}

	n.Status = domain.NotificationStatusSent
	return n, true, nil
}

func (s *notificationProcessingService) HandleDeliveryFailure(ctx context.Context, id uuid.UUID, retryCount int, maxRetries int) (*domain.Notification, error) {
	if retryCount < maxRetries {
		if err := s.service.MarkAsRetrying(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to mark as retrying: %w", err)
		}

		n, err := s.service.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get notification for retry: %w", err)
		}

		newRetryCount := int32(retryCount + 1)
		if err := s.producer.PublishToRetry(ctx, n, newRetryCount); err != nil {
			return nil, fmt.Errorf("failed to publish to retry: %w", err)
		}

		return n, nil
	}

	if err := s.service.MarkAsFailed(ctx, id, "max retries exceeded", retryCount); err != nil {
		return nil, fmt.Errorf("failed to mark as failed: %w", err)
	}

	n, err := s.service.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification after failure: %w", err)
	}

	return n, nil
}

func (s *notificationProcessingService) RecoverStuckNotifications(ctx context.Context) error {
	return s.service.RecoverStuckNotifications(ctx)
}

func (s *notificationProcessingService) PublishDueScheduled(ctx context.Context) error {
	return s.service.PublishDueScheduled(ctx)
}
