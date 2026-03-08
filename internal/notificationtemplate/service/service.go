package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/repository"
)

type NotificationTemplateService interface {
	Create(ctx context.Context, req domain.NotificationTemplateCreateRequest) (*domain.NotificationTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error)
	List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error)
	Update(ctx context.Context, id uuid.UUID, req domain.NotificationTemplateUpdateRequest) (*domain.NotificationTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error)
}

type service struct {
	notificationTemplateRepository repository.NotificationTemplateRepository
}

var _ NotificationTemplateService = (*service)(nil)

func NewNotificationTemplateService(notificationTemplateRepository repository.NotificationTemplateRepository) *service {
	return &service{notificationTemplateRepository: notificationTemplateRepository}
}

func (s *service) Create(ctx context.Context, req domain.NotificationTemplateCreateRequest) (*domain.NotificationTemplate, error) {
	template := &domain.NotificationTemplate{
		Name:    req.Name,
		Channel: req.Channel,
		Content: req.Content,
	}

	if err := s.notificationTemplateRepository.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error) {
	return s.notificationTemplateRepository.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error) {
	return s.notificationTemplateRepository.List(ctx, limit, offset)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req domain.NotificationTemplateUpdateRequest) (*domain.NotificationTemplate, error) {
	template, err := s.notificationTemplateRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Channel != nil {
		template.Channel = *req.Channel
	}
	if req.Content != nil {
		template.Content = *req.Content
	}

	if err := s.notificationTemplateRepository.Update(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.notificationTemplateRepository.Delete(ctx, id)
}

// Render fetches a template by ID and replaces {{key}} placeholders with the provided variables.
// Missing variables leave the placeholder unchanged.
func (s *service) Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error) {
	template, err := s.notificationTemplateRepository.GetByID(ctx, templateID)
	if err != nil {
		return "", err
	}

	content := template.Content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}
