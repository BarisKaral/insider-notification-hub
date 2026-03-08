package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
)

// mockRepository implements NotificationTemplateRepository for testing.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(ctx context.Context, t *domain.NotificationTemplate) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.NotificationTemplate), args.Error(1)
}

func (m *mockRepository) List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*domain.NotificationTemplate), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepository) Update(ctx context.Context, t *domain.NotificationTemplate) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestRender_BasicVariableReplacement(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "Merhaba {{name}}, siparis {{orderId}} kargoda.",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{
		"name":    "Baris",
		"orderId": "123",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Merhaba Baris, siparis 123 kargoda.", result)
	repo.AssertExpectations(t)
}

func TestRender_MultipleVariables(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "{{greeting}} {{name}}, your code is {{code}} and status is {{status}}.",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{
		"greeting": "Hello",
		"name":     "John",
		"code":     "ABC123",
		"status":   "active",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Hello John, your code is ABC123 and status is active.", result)
}

func TestRender_MissingVariable_PlaceholderStays(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "Hello {{name}}, your order {{orderId}} is ready.",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{
		"name": "Baris",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Hello Baris, your order {{orderId}} is ready.", result)
}

func TestRender_EmptyVariables(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "Hello {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{})

	assert.NoError(t, err)
	assert.Equal(t, "Hello {{name}}!", result)
}

func TestRender_NilVariables(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "Hello {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, nil)

	assert.NoError(t, err)
	assert.Equal(t, "Hello {{name}}!", result)
}

func TestRender_TemplateNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(nil, domain.ErrNotificationTemplateNotFound)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
	assert.Empty(t, result)
}

func TestRender_NoPlaceholders(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "This is a static message with no variables.",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.NoError(t, err)
	assert.Equal(t, "This is a static message with no variables.", result)
}

func TestRender_RepeatedPlaceholder(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&domain.NotificationTemplate{
		ID:      templateID,
		Content: "{{name}} says hello. Goodbye, {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.NoError(t, err)
	assert.Equal(t, "Baris says hello. Goodbye, Baris!", result)
}

func TestCreate_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()

	req := domain.NotificationTemplateCreateRequest{
		Name:    "order_shipped",
		Channel: "sms",
		Content: "Your order {{orderId}} has been shipped.",
	}

	repo.On("Create", ctx, mock.AnythingOfType("*domain.NotificationTemplate")).Return(nil)

	tmpl, err := svc.Create(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "order_shipped", tmpl.Name)
	assert.Equal(t, "sms", tmpl.Channel)
	assert.Equal(t, "Your order {{orderId}} has been shipped.", tmpl.Content)
	repo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	existing := &domain.NotificationTemplate{
		ID:      templateID,
		Name:    "old_name",
		Channel: "sms",
		Content: "old content",
	}

	newName := "new_name"
	newContent := "new content"

	repo.On("GetByID", ctx, templateID).Return(existing, nil)
	repo.On("Update", ctx, existing).Return(nil)

	tmpl, err := svc.Update(ctx, templateID, domain.NotificationTemplateUpdateRequest{
		Name:    &newName,
		Content: &newContent,
	})

	assert.NoError(t, err)
	assert.Equal(t, "new_name", tmpl.Name)
	assert.Equal(t, "new content", tmpl.Content)
	assert.Equal(t, "sms", tmpl.Channel)
	repo.AssertExpectations(t)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(nil, domain.ErrNotificationTemplateNotFound)

	newName := "new_name"
	tmpl, err := svc.Update(ctx, templateID, domain.NotificationTemplateUpdateRequest{Name: &newName})

	assert.Nil(t, tmpl)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
}

func TestDTO_CreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.NotificationTemplateCreateRequest
		wantErr bool
	}{
		{
			name:    "valid request",
			req:     domain.NotificationTemplateCreateRequest{Name: "test", Channel: "sms", Content: "hello"},
			wantErr: false,
		},
		{
			name:    "missing name",
			req:     domain.NotificationTemplateCreateRequest{Channel: "sms", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "missing channel",
			req:     domain.NotificationTemplateCreateRequest{Name: "test", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "invalid channel",
			req:     domain.NotificationTemplateCreateRequest{Name: "test", Channel: "fax", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "missing content",
			req:     domain.NotificationTemplateCreateRequest{Name: "test", Channel: "sms"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDTO_UpdateRequest_Validate(t *testing.T) {
	name := "test"
	invalidCh := "fax"

	tests := []struct {
		name    string
		req     domain.NotificationTemplateUpdateRequest
		wantErr bool
	}{
		{
			name:    "valid with name only",
			req:     domain.NotificationTemplateUpdateRequest{Name: &name},
			wantErr: false,
		},
		{
			name:    "no fields provided",
			req:     domain.NotificationTemplateUpdateRequest{},
			wantErr: true,
		},
		{
			name:    "invalid channel",
			req:     domain.NotificationTemplateUpdateRequest{Channel: &invalidCh},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToResponse_MapsAllFields(t *testing.T) {
	id := uuid.New()
	tmpl := &domain.NotificationTemplate{
		ID:      id,
		Name:    "test",
		Channel: "sms",
		Content: "hello",
	}

	resp := domain.ToNotificationTemplateResponse(tmpl)

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "test", resp.Name)
	assert.Equal(t, "sms", resp.Channel)
	assert.Equal(t, "hello", resp.Content)
}

func TestToResponseList_Maps(t *testing.T) {
	templates := []*domain.NotificationTemplate{
		{ID: uuid.New(), Name: "a"},
		{ID: uuid.New(), Name: "b"},
	}

	responses := domain.ToNotificationTemplateResponseList(templates)

	assert.Len(t, responses, 2)
	assert.Equal(t, "a", responses[0].Name)
	assert.Equal(t, "b", responses[1].Name)
}

// --- GetByID tests ---

func TestGetByID_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	expected := &domain.NotificationTemplate{
		ID:      templateID,
		Name:    "order_shipped",
		Channel: "sms",
		Content: "Your order has been shipped.",
	}

	repo.On("GetByID", ctx, templateID).Return(expected, nil)

	result, err := svc.GetByID(ctx, templateID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Equal(t, templateID, result.ID)
	assert.Equal(t, "order_shipped", result.Name)
	assert.Equal(t, "sms", result.Channel)
	assert.Equal(t, "Your order has been shipped.", result.Content)
	repo.AssertExpectations(t)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(nil, domain.ErrNotificationTemplateNotFound)

	result, err := svc.GetByID(ctx, templateID)

	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
	repo.AssertExpectations(t)
}

// --- List tests ---

func TestList_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()

	templates := []*domain.NotificationTemplate{
		{ID: uuid.New(), Name: "template_a", Channel: "sms", Content: "content a"},
		{ID: uuid.New(), Name: "template_b", Channel: "email", Content: "content b"},
		{ID: uuid.New(), Name: "template_c", Channel: "push", Content: "content c"},
	}

	repo.On("List", ctx, 10, 0).Return(templates, int64(3), nil)

	result, total, err := svc.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, "template_a", result[0].Name)
	assert.Equal(t, "template_b", result[1].Name)
	assert.Equal(t, "template_c", result[2].Name)
	repo.AssertExpectations(t)
}

func TestList_Empty(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()

	repo.On("List", ctx, 10, 0).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	result, total, err := svc.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, int64(0), total)
	repo.AssertExpectations(t)
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("Delete", ctx, templateID).Return(nil)

	err := svc.Delete(ctx, templateID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDelete_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("Delete", ctx, templateID).Return(domain.ErrNotificationTemplateNotFound)

	err := svc.Delete(ctx, templateID)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
	repo.AssertExpectations(t)
}

// --- Create error tests ---

func TestCreate_RepositoryError(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()

	req := domain.NotificationTemplateCreateRequest{
		Name:    "duplicate_name",
		Channel: "sms",
		Content: "content",
	}

	repoErr := errors.New("database connection failed")
	repo.On("Create", ctx, mock.AnythingOfType("*domain.NotificationTemplate")).Return(repoErr)

	result, err := svc.Create(ctx, req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, repoErr, err)
	repo.AssertExpectations(t)
}

// --- Update error tests ---

func TestUpdate_RepositoryUpdateError(t *testing.T) {
	repo := new(mockRepository)
	svc := NewNotificationTemplateService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	existing := &domain.NotificationTemplate{
		ID:      templateID,
		Name:    "old_name",
		Channel: "sms",
		Content: "old content",
	}

	newName := "new_name"
	updateErr := errors.New("database write failed")

	repo.On("GetByID", ctx, templateID).Return(existing, nil)
	repo.On("Update", ctx, existing).Return(updateErr)

	result, err := svc.Update(ctx, templateID, domain.NotificationTemplateUpdateRequest{
		Name: &newName,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, updateErr, err)
	repo.AssertExpectations(t)
}
