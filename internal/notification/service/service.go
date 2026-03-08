package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/repository"
	ntService "github.com/baris/notification-hub/internal/notificationtemplate/service"
	"github.com/google/uuid"
)

// NotificationService defines the business logic interface for notifications.
type NotificationService interface {
	Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error)
	CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error)
	List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error)
	Cancel(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	MarkAsProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error
	MarkAsQueued(ctx context.Context, id uuid.UUID) error
	MarkAsRetrying(ctx context.Context, id uuid.UUID) error
}

// NotificationProducer publishes notifications to message queues.
type NotificationProducer interface {
	Publish(ctx context.Context, n *domain.Notification) error
	PublishBatch(ctx context.Context, notifications []*domain.Notification) error
	PublishToRetry(ctx context.Context, n *domain.Notification, retryCount int32) error
}

type notificationService struct {
	notificationRepository      repository.NotificationRepository
	notificationTemplateService ntService.NotificationTemplateService
}

var _ NotificationService = (*notificationService)(nil)

// NewNotificationService creates a new NotificationService.
func NewNotificationService(notificationRepository repository.NotificationRepository, notificationTemplateService ntService.NotificationTemplateService) *notificationService {
	return &notificationService{
		notificationRepository:      notificationRepository,
		notificationTemplateService: notificationTemplateService,
	}
}

func (s *notificationService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	// Determine idempotency key.
	var key string
	if idempotencyKey != nil {
		key = *idempotencyKey
		// Check for duplicate.
		existing, err := s.notificationRepository.GetByIdempotencyKey(ctx, key)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, domain.ErrNotificationDuplicateIdempotencyKey
		}
	} else {
		generated := uuid.New().String()
		key = generated
	}

	// Resolve content: either direct or via template rendering.
	content := ""
	var templateID *uuid.UUID
	var templateVars json.RawMessage

	if req.TemplateID != nil {
		rendered, err := s.notificationTemplateService.Render(ctx, *req.TemplateID, req.Variables)
		if err != nil {
			return nil, err
		}
		content = rendered
		templateID = req.TemplateID

		if req.Variables != nil {
			varsJSON, err := json.Marshal(req.Variables)
			if err != nil {
				return nil, err
			}
			templateVars = varsJSON
		}
	} else if req.Content != nil {
		content = *req.Content
	}

	// Determine status based on scheduling.
	status := domain.NotificationStatusPending
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now().UTC()) {
		status = domain.NotificationStatusScheduled
	}

	// Default priority.
	priority := domain.NotificationPriorityNormal
	if req.Priority != "" {
		priority = domain.NotificationPriority(req.Priority)
	}

	notification := &domain.Notification{
		Recipient:      req.Recipient,
		Channel:        domain.NotificationChannel(req.Channel),
		Content:        content,
		Priority:       priority,
		Status:         status,
		IdempotencyKey: &key,
		TemplateID:     templateID,
		TemplateVars:   templateVars,
		ScheduledAt:    req.ScheduledAt,
	}

	if err := s.notificationRepository.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *notificationService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	batchID := uuid.New()
	notifications := make([]*domain.Notification, 0, len(req.Notifications))

	for _, request := range req.Notifications {
		// Resolve content.
		content := ""
		var templateID *uuid.UUID
		var templateVars json.RawMessage

		if request.TemplateID != nil {
			rendered, err := s.notificationTemplateService.Render(ctx, *request.TemplateID, request.Variables)
			if err != nil {
				return nil, uuid.Nil, err
			}
			content = rendered
			templateID = request.TemplateID

			if request.Variables != nil {
				varsJSON, err := json.Marshal(request.Variables)
				if err != nil {
					return nil, uuid.Nil, err
				}
				templateVars = varsJSON
			}
		} else if request.Content != nil {
			content = *request.Content
		}

		// Determine status.
		status := domain.NotificationStatusPending
		if request.ScheduledAt != nil && request.ScheduledAt.After(time.Now().UTC()) {
			status = domain.NotificationStatusScheduled
		}

		// Default priority.
		priority := domain.NotificationPriorityNormal
		if request.Priority != "" {
			priority = domain.NotificationPriority(request.Priority)
		}

		// Server-generated idempotency key for each notification in batch.
		key := uuid.New().String()

		notification := &domain.Notification{
			Recipient:      request.Recipient,
			Channel:        domain.NotificationChannel(request.Channel),
			Content:        content,
			Priority:       priority,
			Status:         status,
			BatchID:        &batchID,
			IdempotencyKey: &key,
			TemplateID:     templateID,
			TemplateVars:   templateVars,
			ScheduledAt:    request.ScheduledAt,
		}

		notifications = append(notifications, notification)
	}

	if err := s.notificationRepository.CreateBatch(ctx, notifications); err != nil {
		return nil, uuid.Nil, err
	}

	return notifications, batchID, nil
}

func (s *notificationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	return s.notificationRepository.GetByID(ctx, id)
}

func (s *notificationService) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	return s.notificationRepository.GetByBatchID(ctx, batchID)
}

func (s *notificationService) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	return s.notificationRepository.List(ctx, filter)
}

func (s *notificationService) Cancel(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	notification, err := s.notificationRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch notification.Status {
	case domain.NotificationStatusSent:
		return nil, domain.ErrNotificationAlreadySent
	case domain.NotificationStatusCancelled:
		return nil, domain.ErrNotificationAlreadyCancelled
	case domain.NotificationStatusProcessing, domain.NotificationStatusRetrying:
		return nil, domain.ErrNotificationCancelFailed
	case domain.NotificationStatusPending, domain.NotificationStatusScheduled, domain.NotificationStatusQueued, domain.NotificationStatusFailed:
		notification.Status = domain.NotificationStatusCancelled
		if err := s.notificationRepository.Update(ctx, notification); err != nil {
			return nil, err
		}
		return notification, nil
	default:
		return nil, domain.ErrNotificationCancelFailed
	}
}

func (s *notificationService) MarkAsProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	notification, err := s.notificationRepository.GetForProcessing(ctx, id)
	if err != nil {
		return nil, err
	}

	// Skip cancelled or sent — return as-is so consumer can check status.
	if notification.Status == domain.NotificationStatusCancelled || notification.Status == domain.NotificationStatusSent {
		return notification, nil
	}

	if !notification.Status.CanTransitionTo(domain.NotificationStatusProcessing) {
		return nil, domain.ErrNotificationInvalidStatus
	}

	notification.Status = domain.NotificationStatusProcessing
	if err := s.notificationRepository.Update(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *notificationService) MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error {
	notification, err := s.notificationRepository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	notification.Status = domain.NotificationStatusSent
	notification.ProviderMsgID = &providerMsgID
	notification.SentAt = &now

	return s.notificationRepository.Update(ctx, notification)
}

func (s *notificationService) MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error {
	notification, err := s.notificationRepository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	notification.Status = domain.NotificationStatusFailed
	notification.FailureReason = &reason
	notification.FailedAt = &now
	notification.RetryCount = retryCount

	return s.notificationRepository.Update(ctx, notification)
}

func (s *notificationService) MarkAsQueued(ctx context.Context, id uuid.UUID) error {
	notification, err := s.notificationRepository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	notification.Status = domain.NotificationStatusQueued
	return s.notificationRepository.Update(ctx, notification)
}

func (s *notificationService) MarkAsRetrying(ctx context.Context, id uuid.UUID) error {
	notification, err := s.notificationRepository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	notification.Status = domain.NotificationStatusRetrying

	return s.notificationRepository.Update(ctx, notification)
}

