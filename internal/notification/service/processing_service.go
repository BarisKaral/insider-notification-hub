package service

import (
	"context"
	"fmt"
	"time"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/provider"
	"github.com/baris/notification-hub/internal/notification/repository"
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
	notificationService    NotificationService
	notificationProducer   NotificationProducer
	providers              map[domain.NotificationChannel]provider.NotificationProvider
	notificationRepository repository.NotificationRepository
}

var _ NotificationProcessingService = (*notificationProcessingService)(nil)

// NewNotificationProcessingService creates a new NotificationProcessingService.
func NewNotificationProcessingService(
	notificationService NotificationService,
	notificationProducer NotificationProducer,
	providers map[domain.NotificationChannel]provider.NotificationProvider,
	notificationRepository repository.NotificationRepository,
) NotificationProcessingService {
	return &notificationProcessingService{
		notificationService:    notificationService,
		notificationProducer:   notificationProducer,
		providers:              providers,
		notificationRepository: notificationRepository,
	}
}

func (s *notificationProcessingService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	notification, err := s.notificationService.Create(ctx, req, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if notification.Status != domain.NotificationStatusScheduled {
		if err := s.notificationProducer.Publish(ctx, notification); err != nil {
			logger.Error().Err(err).Str("notificationID", notification.ID.String()).Msg("failed to publish notification")
			return notification, nil
		}

		if err := s.notificationService.MarkAsQueued(ctx, notification.ID); err != nil {
			logger.Error().Err(err).Str("notificationID", notification.ID.String()).Msg("failed to mark notification as queued")
			return notification, nil
		}

		notification.Status = domain.NotificationStatusQueued
	}

	return notification, nil
}

func (s *notificationProcessingService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	notifications, batchID, err := s.notificationService.CreateBatch(ctx, req)
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
		if err := s.notificationProducer.PublishBatch(ctx, toPublish); err != nil {
			logger.Error().Err(err).Str("batchID", batchID.String()).Msg("failed to publish batch notifications")
			return notifications, batchID, nil
		}

		for _, n := range toPublish {
			if err := s.notificationService.MarkAsQueued(ctx, n.ID); err != nil {
				logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to mark notification as queued")
			} else {
				n.Status = domain.NotificationStatusQueued
			}
		}
	}

	return notifications, batchID, nil
}

func (s *notificationProcessingService) ProcessAndSend(ctx context.Context, id uuid.UUID) (*domain.Notification, bool, error) {
	notification, err := s.notificationService.MarkAsProcessing(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if notification.Status == domain.NotificationStatusCancelled || notification.Status == domain.NotificationStatusSent {
		return notification, false, nil
	}

	notificationProvider, ok := s.providers[notification.Channel]
	if !ok {
		return nil, false, fmt.Errorf("no provider for channel %s", notification.Channel)
	}

	response, err := notificationProvider.Send(ctx, &provider.ProviderRequest{
		To:      notification.Recipient,
		Channel: string(notification.Channel),
		Content: notification.Content,
	})
	if err != nil {
		return nil, false, err
	}

	if err := s.notificationService.MarkAsSent(ctx, id, response.MessageID); err != nil {
		return nil, false, err
	}

	notification.Status = domain.NotificationStatusSent
	return notification, true, nil
}

func (s *notificationProcessingService) HandleDeliveryFailure(ctx context.Context, id uuid.UUID, retryCount int, maxRetries int) (*domain.Notification, error) {
	if retryCount < maxRetries {
		if err := s.notificationService.MarkAsRetrying(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to mark as retrying: %w", err)
		}

		notification, err := s.notificationService.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get notification for retry: %w", err)
		}

		newRetryCount := int32(retryCount + 1)
		if err := s.notificationProducer.PublishToRetry(ctx, notification, newRetryCount); err != nil {
			return nil, fmt.Errorf("failed to publish to retry: %w", err)
		}

		return notification, nil
	}

	if err := s.notificationService.MarkAsFailed(ctx, id, "max retries exceeded", retryCount); err != nil {
		return nil, fmt.Errorf("failed to mark as failed: %w", err)
	}

	notification, err := s.notificationService.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification after failure: %w", err)
	}

	return notification, nil
}

func (s *notificationProcessingService) RecoverStuckNotifications(ctx context.Context) error {
	notifications, err := s.notificationRepository.GetRecoverableNotifications(ctx, 30*time.Second)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get recoverable notifications")
		return nil
	}

	for _, n := range notifications {
		if err := s.notificationProducer.Publish(ctx, n); err != nil {
			logger.Error().Err(err).Str("notificationId", n.ID.String()).Msg("failed to publish stuck notification")
			continue
		}

		if err := s.notificationService.MarkAsQueued(ctx, n.ID); err != nil {
			logger.Error().Err(err).Str("notificationId", n.ID.String()).Msg("failed to update stuck notification status")
			continue
		}

		logger.Info().Str("notificationId", n.ID.String()).Msg("recovered stuck notification")
	}

	return nil
}

func (s *notificationProcessingService) PublishDueScheduled(ctx context.Context) error {
	notifications, err := s.notificationRepository.GetDueScheduledNotifications(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get due scheduled notifications")
		return nil
	}

	for _, n := range notifications {
		if err := s.notificationProducer.Publish(ctx, n); err != nil {
			logger.Error().Err(err).Str("notificationId", n.ID.String()).Msg("failed to publish scheduled notification")
			continue
		}

		if err := s.notificationService.MarkAsQueued(ctx, n.ID); err != nil {
			logger.Error().Err(err).Str("notificationId", n.ID.String()).Msg("failed to update scheduled notification status")
			continue
		}

		logger.Info().Str("notificationId", n.ID.String()).Msg("published due scheduled notification")
	}

	return nil
}
