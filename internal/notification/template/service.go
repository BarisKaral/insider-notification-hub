package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type TemplateService interface {
	Create(ctx context.Context, req TemplateCreateRequest) (*Template, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Template, error)
	List(ctx context.Context, limit, offset int) ([]*Template, int64, error)
	Update(ctx context.Context, id uuid.UUID, req TemplateUpdateRequest) (*Template, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error)
}

type service struct {
	repo TemplateRepository
}

var _ TemplateService = (*service)(nil)

func NewTemplateService(repo TemplateRepository) TemplateService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req TemplateCreateRequest) (*Template, error) {
	t := &Template{
		Name:    req.Name,
		Channel: req.Channel,
		Content: req.Content,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*Template, int64, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req TemplateUpdateRequest) (*Template, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Channel != nil {
		t.Channel = *req.Channel
	}
	if req.Content != nil {
		t.Content = *req.Content
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Render fetches a template by ID and replaces {{key}} placeholders with the provided variables.
// Missing variables leave the placeholder unchanged.
func (s *service) Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error) {
	t, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return "", err
	}

	content := t.Content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}
