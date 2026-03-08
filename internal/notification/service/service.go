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
	repo            repository.NotificationRepository
	templateService ntService.NotificationTemplateService
}

var _ NotificationService = (*notificationService)(nil)

// NewNotificationService creates a new NotificationService.
func NewNotificationService(repo repository.NotificationRepository, templateSvc ntService.NotificationTemplateService) *notificationService {
	return &notificationService{
		repo:            repo,
		templateService: templateSvc,
	}
}

func (s *notificationService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	// Determine idempotency key.
	var key string
	if idempotencyKey != nil {
		key = *idempotencyKey
		// Check for duplicate.
		existing, err := s.repo.GetByIdempotencyKey(ctx, key)
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
		rendered, err := s.templateService.Render(ctx, *req.TemplateID, req.Variables)
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

	n := &domain.Notification{
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

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

func (s *notificationService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	batchID := uuid.New()
	notifications := make([]*domain.Notification, 0, len(req.Notifications))

	for _, r := range req.Notifications {
		// Resolve content.
		content := ""
		var templateID *uuid.UUID
		var templateVars json.RawMessage

		if r.TemplateID != nil {
			rendered, err := s.templateService.Render(ctx, *r.TemplateID, r.Variables)
			if err != nil {
				return nil, uuid.Nil, err
			}
			content = rendered
			templateID = r.TemplateID

			if r.Variables != nil {
				varsJSON, err := json.Marshal(r.Variables)
				if err != nil {
					return nil, uuid.Nil, err
				}
				templateVars = varsJSON
			}
		} else if r.Content != nil {
			content = *r.Content
		}

		// Determine status.
		status := domain.NotificationStatusPending
		if r.ScheduledAt != nil && r.ScheduledAt.After(time.Now().UTC()) {
			status = domain.NotificationStatusScheduled
		}

		// Default priority.
		priority := domain.NotificationPriorityNormal
		if r.Priority != "" {
			priority = domain.NotificationPriority(r.Priority)
		}

		// Server-generated idempotency key for each notification in batch.
		key := uuid.New().String()

		n := &domain.Notification{
			Recipient:      r.Recipient,
			Channel:        domain.NotificationChannel(r.Channel),
			Content:        content,
			Priority:       priority,
			Status:         status,
			BatchID:        &batchID,
			IdempotencyKey: &key,
			TemplateID:     templateID,
			TemplateVars:   templateVars,
			ScheduledAt:    r.ScheduledAt,
		}

		notifications = append(notifications, n)
	}

	if err := s.repo.CreateBatch(ctx, notifications); err != nil {
		return nil, uuid.Nil, err
	}

	return notifications, batchID, nil
}

func (s *notificationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *notificationService) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	return s.repo.GetByBatchID(ctx, batchID)
}

func (s *notificationService) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *notificationService) Cancel(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch n.Status {
	case domain.NotificationStatusSent:
		return nil, domain.ErrNotificationAlreadySent
	case domain.NotificationStatusCancelled:
		return nil, domain.ErrNotificationAlreadyCancelled
	case domain.NotificationStatusProcessing, domain.NotificationStatusRetrying:
		return nil, domain.ErrNotificationCancelFailed
	case domain.NotificationStatusPending, domain.NotificationStatusScheduled, domain.NotificationStatusQueued, domain.NotificationStatusFailed:
		n.Status = domain.NotificationStatusCancelled
		if err := s.repo.Update(ctx, n); err != nil {
			return nil, err
		}
		return n, nil
	default:
		return nil, domain.ErrNotificationCancelFailed
	}
}

func (s *notificationService) MarkAsProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	n, err := s.repo.GetForProcessing(ctx, id)
	if err != nil {
		return nil, err
	}

	// Skip cancelled or sent — return as-is so consumer can check status.
	if n.Status == domain.NotificationStatusCancelled || n.Status == domain.NotificationStatusSent {
		return n, nil
	}

	if !n.Status.CanTransitionTo(domain.NotificationStatusProcessing) {
		return nil, domain.ErrNotificationInvalidStatus
	}

	n.Status = domain.NotificationStatusProcessing
	if err := s.repo.Update(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

func (s *notificationService) MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	n.Status = domain.NotificationStatusSent
	n.ProviderMsgID = &providerMsgID
	n.SentAt = &now

	return s.repo.Update(ctx, n)
}

func (s *notificationService) MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	n.Status = domain.NotificationStatusFailed
	n.FailureReason = &reason
	n.FailedAt = &now
	n.RetryCount = retryCount

	return s.repo.Update(ctx, n)
}

func (s *notificationService) MarkAsQueued(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	n.Status = domain.NotificationStatusQueued
	return s.repo.Update(ctx, n)
}

func (s *notificationService) MarkAsRetrying(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	n.Status = domain.NotificationStatusRetrying

	return s.repo.Update(ctx, n)
}

