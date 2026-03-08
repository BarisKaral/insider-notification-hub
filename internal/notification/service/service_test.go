package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bariskaral/insider-notification-hub/internal/notification/domain"
	ntDomain "github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockNotificationRepository struct {
	mock.Mock
}

func (m *mockNotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *mockNotificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	args := m.Called(ctx, notifications)
	return args.Error(0)
}

func (m *mockNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	args := m.Called(ctx, batchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *mockNotificationRepository) Update(ctx context.Context, n *domain.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *mockNotificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) GetForProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*domain.Notification, error) {
	args := m.Called(ctx, staleDuration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) GetDueScheduledNotifications(ctx context.Context) ([]*domain.Notification, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

type mockTemplateService struct {
	mock.Mock
}

func (m *mockTemplateService) Create(ctx context.Context, req ntDomain.NotificationTemplateCreateRequest) (*ntDomain.NotificationTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ntDomain.NotificationTemplate), args.Error(1)
}

func (m *mockTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*ntDomain.NotificationTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ntDomain.NotificationTemplate), args.Error(1)
}

func (m *mockTemplateService) List(ctx context.Context, limit, offset int) ([]*ntDomain.NotificationTemplate, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*ntDomain.NotificationTemplate), args.Get(1).(int64), args.Error(2)
}

func (m *mockTemplateService) Update(ctx context.Context, id uuid.UUID, req ntDomain.NotificationTemplateUpdateRequest) (*ntDomain.NotificationTemplate, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ntDomain.NotificationTemplate), args.Error(1)
}

func (m *mockTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTemplateService) Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error) {
	args := m.Called(ctx, templateID, variables)
	return args.String(0), args.Error(1)
}

// --- Tests ---

func newTestService(repo *mockNotificationRepository, tmplSvc *mockTemplateService) NotificationService {
	return NewNotificationService(repo, tmplSvc)
}

// --- Create ---

func TestNotificationService_Create_DirectContent(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello World"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
		Priority:  "high",
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).
		Run(func(args mock.Arguments) {
			n := args.Get(1).(*domain.Notification)
			n.ID = uuid.New()
		}).
		Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, "Hello World", result.Content)
	assert.Equal(t, domain.NotificationChannelSMS, result.Channel)
	assert.Equal(t, domain.NotificationPriorityHigh, result.Priority)
	assert.Equal(t, domain.NotificationStatusPending, result.Status)
	assert.Equal(t, "+1234567890", result.Recipient)
	assert.NotNil(t, result.IdempotencyKey)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_WithTemplate(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	variables := map[string]string{"name": "John"}
	req := domain.NotificationCreateRequest{
		Recipient:  "john@example.com",
		Channel:    "email",
		TemplateID: &templateID,
		Variables:  variables,
	}

	tmplSvc.On("Render", mock.Anything, templateID, variables).Return("Hello John", nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, "Hello John", result.Content)
	assert.Equal(t, &templateID, result.TemplateID)
	assert.NotNil(t, result.TemplateVars)

	var storedVars map[string]string
	err = json.Unmarshal(result.TemplateVars, &storedVars)
	require.NoError(t, err)
	assert.Equal(t, "John", storedVars["name"])

	tmplSvc.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_WithTemplateNilVariables(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	req := domain.NotificationCreateRequest{
		Recipient:  "john@example.com",
		Channel:    "email",
		TemplateID: &templateID,
		Variables:  nil, // nil variables
	}

	tmplSvc.On("Render", mock.Anything, templateID, (map[string]string)(nil)).Return("Hello default", nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, "Hello default", result.Content)
	assert.Equal(t, &templateID, result.TemplateID)
	// templateVars should be nil since Variables was nil
	assert.Nil(t, result.TemplateVars)

	tmplSvc.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_WithTemplateRenderFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	variables := map[string]string{"name": "John"}
	req := domain.NotificationCreateRequest{
		Recipient:  "john@example.com",
		Channel:    "email",
		TemplateID: &templateID,
		Variables:  variables,
	}

	renderErr := errors.New("template not found")
	tmplSvc.On("Render", mock.Anything, templateID, variables).Return("", renderErr)

	result, err := svc.Create(context.Background(), req, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, renderErr)
	// repo.Create should never be called
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	tmplSvc.AssertExpectations(t)
}

func TestNotificationService_Create_DuplicateIdempotencyKey(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	key := "my-unique-key"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	existing := &domain.Notification{ID: uuid.New()}
	repo.On("GetByIdempotencyKey", mock.Anything, key).Return(existing, nil)

	result, err := svc.Create(context.Background(), req, &key)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationDuplicateIdempotencyKey)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_IdempotencyKeyLookupError(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	key := "my-key"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	dbErr := errors.New("database connection failed")
	repo.On("GetByIdempotencyKey", mock.Anything, key).Return(nil, dbErr)

	result, err := svc.Create(context.Background(), req, &key)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, dbErr)
	// repo.Create should never be called
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_ScheduledAtFuture(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Scheduled message"
	futureTime := time.Now().UTC().Add(24 * time.Hour)
	req := domain.NotificationCreateRequest{
		Recipient:   "+1234567890",
		Channel:     "sms",
		Content:     &content,
		ScheduledAt: &futureTime,
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusScheduled, result.Status)
	assert.NotNil(t, result.ScheduledAt)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_ScheduledAtPast(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Past scheduled message"
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	req := domain.NotificationCreateRequest{
		Recipient:   "+1234567890",
		Channel:     "sms",
		Content:     &content,
		ScheduledAt: &pastTime,
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	// Past scheduled_at should result in pending, not scheduled
	assert.Equal(t, domain.NotificationStatusPending, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_DefaultPriority(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
		// No priority specified
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationPriorityNormal, result.Priority)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_RepoCreateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	repoErr := errors.New("insert failed")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(repoErr)

	result, err := svc.Create(context.Background(), req, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_UniqueIdempotencyKey(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	key := "unique-key-123"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	// Key not found — proceed with creation
	repo.On("GetByIdempotencyKey", mock.Anything, key).Return(nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, &key)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, key, *result.IdempotencyKey)
	repo.AssertExpectations(t)
}

func TestNotificationService_Create_NoContentNoTemplate(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		// No Content, no TemplateID
	}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Create(context.Background(), req, nil)

	require.NoError(t, err)
	// Content should be empty string
	assert.Equal(t, "", result.Content)
	repo.AssertExpectations(t)
}

// --- CreateBatch ---

func TestNotificationService_CreateBatch(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content1 := "Hello 1"
	content2 := "Hello 2"
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content1},
			{Recipient: "+222", Channel: "sms", Content: &content2},
		},
	}

	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	results, batchID, err := svc.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotEqual(t, uuid.Nil, batchID)

	// All notifications share the same batchID
	for _, n := range results {
		assert.NotNil(t, n.BatchID)
		assert.Equal(t, batchID, *n.BatchID)
		assert.NotNil(t, n.IdempotencyKey)
		assert.Equal(t, domain.NotificationStatusPending, n.Status)
	}

	assert.Equal(t, "Hello 1", results[0].Content)
	assert.Equal(t, "Hello 2", results[1].Content)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_WithTemplate(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	vars1 := map[string]string{"name": "Alice"}
	vars2 := map[string]string{"name": "Bob"}
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "alice@example.com", Channel: "email", TemplateID: &templateID, Variables: vars1},
			{Recipient: "bob@example.com", Channel: "email", TemplateID: &templateID, Variables: vars2},
		},
	}

	tmplSvc.On("Render", mock.Anything, templateID, vars1).Return("Hello Alice", nil)
	tmplSvc.On("Render", mock.Anything, templateID, vars2).Return("Hello Bob", nil)
	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	results, batchID, err := svc.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotEqual(t, uuid.Nil, batchID)
	assert.Equal(t, "Hello Alice", results[0].Content)
	assert.Equal(t, "Hello Bob", results[1].Content)
	assert.Equal(t, &templateID, results[0].TemplateID)
	assert.NotNil(t, results[0].TemplateVars)
	assert.NotNil(t, results[1].TemplateVars)

	tmplSvc.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_WithTemplateNilVariables(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "alice@example.com", Channel: "email", TemplateID: &templateID, Variables: nil},
		},
	}

	tmplSvc.On("Render", mock.Anything, templateID, (map[string]string)(nil)).Return("Hello default", nil)
	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	results, batchID, err := svc.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotEqual(t, uuid.Nil, batchID)
	assert.Equal(t, "Hello default", results[0].Content)
	assert.Nil(t, results[0].TemplateVars)

	tmplSvc.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_TemplateRenderFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	templateID := uuid.New()
	vars := map[string]string{"name": "Alice"}
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "alice@example.com", Channel: "email", TemplateID: &templateID, Variables: vars},
		},
	}

	renderErr := errors.New("template render failed")
	tmplSvc.On("Render", mock.Anything, templateID, vars).Return("", renderErr)

	results, batchID, err := svc.CreateBatch(context.Background(), req)

	assert.Nil(t, results)
	assert.Equal(t, uuid.Nil, batchID)
	assert.ErrorIs(t, err, renderErr)
	repo.AssertNotCalled(t, "CreateBatch", mock.Anything, mock.Anything)
	tmplSvc.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_RepoError(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content},
		},
	}

	repoErr := errors.New("batch insert failed")
	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(repoErr)

	results, batchID, err := svc.CreateBatch(context.Background(), req)

	assert.Nil(t, results)
	assert.Equal(t, uuid.Nil, batchID)
	assert.ErrorIs(t, err, repoErr)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_ScheduledAtFuture(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Scheduled"
	futureTime := time.Now().UTC().Add(24 * time.Hour)
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content, ScheduledAt: &futureTime},
		},
	}

	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	results, _, err := svc.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusScheduled, results[0].Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateBatch_CustomPriority(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	content := "Hello"
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content, Priority: "high"},
		},
	}

	repo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]*domain.Notification")).Return(nil)

	results, _, err := svc.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationPriorityHigh, results[0].Priority)
	repo.AssertExpectations(t)
}

// --- Cancel ---

func TestNotificationService_Cancel_FromPending(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusPending,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Cancel(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromScheduled(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusScheduled,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Cancel(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromQueued(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusQueued,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Cancel(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromFailed(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusFailed,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.Cancel(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromSent(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusSent,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationAlreadySent)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromCancelled(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusCancelled,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationAlreadyCancelled)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromProcessing(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusProcessing,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationCancelFailed)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_FromRetrying(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusRetrying,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationCancelFailed)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_GetByIDFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_Cancel_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusPending,
	}

	updateErr := errors.New("update failed")
	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	result, err := svc.Cancel(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

// --- MarkAsProcessing ---

func TestNotificationService_MarkAsProcessing_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusQueued,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusProcessing, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_SkipsCancelled(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusCancelled,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	// Update should NOT be called — notification is returned as-is
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_SkipsSent(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusSent,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusSent, result.Status)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_InvalidTransition(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	// "failed" cannot transition directly to "processing"
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusFailed,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationInvalidStatus)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_GetForProcessingFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetForProcessing", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusQueued,
	}

	updateErr := errors.New("update failed")
	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_FromRetrying(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusRetrying,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusProcessing, result.Status)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsProcessing_FromPendingInvalid(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	// "pending" cannot transition to "processing" directly (must go through "queued")
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusPending,
	}

	repo.On("GetForProcessing", mock.Anything, id).Return(existing, nil)

	result, err := svc.MarkAsProcessing(context.Background(), id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotificationInvalidStatus)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

// --- MarkAsSent ---

func TestNotificationService_MarkAsSent(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	providerMsgID := "provider-123"
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusProcessing,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Status == domain.NotificationStatusSent &&
			n.ProviderMsgID != nil && *n.ProviderMsgID == providerMsgID &&
			n.SentAt != nil
	})).Return(nil)

	err := svc.MarkAsSent(context.Background(), id, providerMsgID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsSent_GetByIDFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	err := svc.MarkAsSent(context.Background(), id, "provider-123")

	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsSent_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusProcessing,
	}

	updateErr := errors.New("update failed")
	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	err := svc.MarkAsSent(context.Background(), id, "provider-123")

	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

// --- MarkAsFailed ---

func TestNotificationService_MarkAsFailed(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	reason := "provider timeout"
	retryCount := 3
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusProcessing,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Status == domain.NotificationStatusFailed &&
			n.FailureReason != nil && *n.FailureReason == reason &&
			n.FailedAt != nil &&
			n.RetryCount == retryCount
	})).Return(nil)

	err := svc.MarkAsFailed(context.Background(), id, reason, retryCount)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsFailed_GetByIDFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	err := svc.MarkAsFailed(context.Background(), id, "reason", 1)

	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsFailed_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusProcessing,
	}

	updateErr := errors.New("update failed")
	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	err := svc.MarkAsFailed(context.Background(), id, "reason", 1)

	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

// --- MarkAsQueued ---

func TestNotificationService_MarkAsQueued(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusPending,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Status == domain.NotificationStatusQueued
	})).Return(nil)

	err := svc.MarkAsQueued(context.Background(), id)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsQueued_GetByIDFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	err := svc.MarkAsQueued(context.Background(), id)

	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsQueued_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusPending,
	}

	updateErr := errors.New("update failed")
	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	err := svc.MarkAsQueued(context.Background(), id)

	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

// --- MarkAsRetrying ---

func TestNotificationService_MarkAsRetrying(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusFailed,
	}

	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Status == domain.NotificationStatusRetrying
	})).Return(nil)

	err := svc.MarkAsRetrying(context.Background(), id)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsRetrying_GetByIDFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationNotFound)

	err := svc.MarkAsRetrying(context.Background(), id)

	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsRetrying_UpdateFails(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	existing := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusFailed,
	}

	updateErr := errors.New("update failed")
	repo.On("GetByID", mock.Anything, id).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(updateErr)

	err := svc.MarkAsRetrying(context.Background(), id)

	assert.ErrorIs(t, err, updateErr)
	repo.AssertExpectations(t)
}

// --- GetByID ---

func TestNotificationService_GetByID(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	id := uuid.New()
	expected := &domain.Notification{ID: id, Recipient: "test@example.com"}

	repo.On("GetByID", mock.Anything, id).Return(expected, nil)

	result, err := svc.GetByID(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

// --- GetByBatchID ---

func TestNotificationService_GetByBatchID(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	batchID := uuid.New()
	expected := []*domain.Notification{
		{ID: uuid.New(), BatchID: &batchID},
		{ID: uuid.New(), BatchID: &batchID},
	}

	repo.On("GetByBatchID", mock.Anything, batchID).Return(expected, nil)

	results, err := svc.GetByBatchID(context.Background(), batchID)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	repo.AssertExpectations(t)
}

// --- List ---

func TestNotificationService_List(t *testing.T) {
	repo := new(mockNotificationRepository)
	tmplSvc := new(mockTemplateService)
	svc := newTestService(repo, tmplSvc)

	filter := domain.NotificationListFilter{Limit: 10, Offset: 0}
	expected := []*domain.Notification{{ID: uuid.New()}}

	repo.On("List", mock.Anything, filter).Return(expected, int64(1), nil)

	results, total, err := svc.List(context.Background(), filter)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(1), total)
	repo.AssertExpectations(t)
}
