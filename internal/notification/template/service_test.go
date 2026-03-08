package template

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(ctx context.Context, t *Template) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *mockRepository) List(ctx context.Context, limit, offset int) ([]*Template, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*Template), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepository) Update(ctx context.Context, t *Template) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestRender_BasicVariableReplacement(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
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
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
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
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
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
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
		ID:      templateID,
		Content: "Hello {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{})

	assert.NoError(t, err)
	assert.Equal(t, "Hello {{name}}!", result)
}

func TestRender_NilVariables(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
		ID:      templateID,
		Content: "Hello {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, nil)

	assert.NoError(t, err)
	assert.Equal(t, "Hello {{name}}!", result)
}

func TestRender_TemplateNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(nil, ErrTemplateNotFound)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.Error(t, err)
	assert.Equal(t, ErrTemplateNotFound, err)
	assert.Empty(t, result)
}

func TestRender_NoPlaceholders(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
		ID:      templateID,
		Content: "This is a static message with no variables.",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.NoError(t, err)
	assert.Equal(t, "This is a static message with no variables.", result)
}

func TestRender_RepeatedPlaceholder(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(&Template{
		ID:      templateID,
		Content: "{{name}} says hello. Goodbye, {{name}}!",
	}, nil)

	result, err := svc.Render(ctx, templateID, map[string]string{"name": "Baris"})

	assert.NoError(t, err)
	assert.Equal(t, "Baris says hello. Goodbye, Baris!", result)
}

func TestCreate_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	req := CreateRequest{
		Name:    "order_shipped",
		Channel: "sms",
		Content: "Your order {{orderId}} has been shipped.",
	}

	repo.On("Create", ctx, mock.AnythingOfType("*template.Template")).Return(nil)

	tmpl, err := svc.Create(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "order_shipped", tmpl.Name)
	assert.Equal(t, "sms", tmpl.Channel)
	assert.Equal(t, "Your order {{orderId}} has been shipped.", tmpl.Content)
	repo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	existing := &Template{
		ID:      templateID,
		Name:    "old_name",
		Channel: "sms",
		Content: "old content",
	}

	newName := "new_name"
	newContent := "new content"

	repo.On("GetByID", ctx, templateID).Return(existing, nil)
	repo.On("Update", ctx, existing).Return(nil)

	tmpl, err := svc.Update(ctx, templateID, UpdateRequest{
		Name:    &newName,
		Content: &newContent,
	})

	assert.NoError(t, err)
	assert.Equal(t, "new_name", tmpl.Name)
	assert.Equal(t, "new content", tmpl.Content)
	assert.Equal(t, "sms", tmpl.Channel) // unchanged
	repo.AssertExpectations(t)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	templateID := uuid.New()

	repo.On("GetByID", ctx, templateID).Return(nil, ErrTemplateNotFound)

	newName := "new_name"
	tmpl, err := svc.Update(ctx, templateID, UpdateRequest{Name: &newName})

	assert.Nil(t, tmpl)
	assert.Equal(t, ErrTemplateNotFound, err)
}

func TestDTO_CreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRequest
		wantErr bool
	}{
		{
			name:    "valid request",
			req:     CreateRequest{Name: "test", Channel: "sms", Content: "hello"},
			wantErr: false,
		},
		{
			name:    "missing name",
			req:     CreateRequest{Channel: "sms", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "missing channel",
			req:     CreateRequest{Name: "test", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "invalid channel",
			req:     CreateRequest{Name: "test", Channel: "fax", Content: "hello"},
			wantErr: true,
		},
		{
			name:    "missing content",
			req:     CreateRequest{Name: "test", Channel: "sms"},
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
		req     UpdateRequest
		wantErr bool
	}{
		{
			name:    "valid with name only",
			req:     UpdateRequest{Name: &name},
			wantErr: false,
		},
		{
			name:    "no fields provided",
			req:     UpdateRequest{},
			wantErr: true,
		},
		{
			name:    "invalid channel",
			req:     UpdateRequest{Channel: &invalidCh},
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
	tmpl := &Template{
		ID:      id,
		Name:    "test",
		Channel: "sms",
		Content: "hello",
	}

	resp := ToResponse(tmpl)

	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "test", resp.Name)
	assert.Equal(t, "sms", resp.Channel)
	assert.Equal(t, "hello", resp.Content)
}

func TestToResponseList_Maps(t *testing.T) {
	templates := []*Template{
		{ID: uuid.New(), Name: "a"},
		{ID: uuid.New(), Name: "b"},
	}

	responses := ToResponseList(templates)

	assert.Len(t, responses, 2)
	assert.Equal(t, "a", responses[0].Name)
	assert.Equal(t, "b", responses[1].Name)
}
