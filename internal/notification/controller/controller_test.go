package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bariskaral/insider-notification-hub/internal/notification/domain"
	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/bariskaral/insider-notification-hub/pkg/response"
)

// --- Service Mock ---

type mockNotificationService struct {
	mock.Mock
}

func (m *mockNotificationService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	args := m.Called(ctx, req, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(uuid.UUID), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(uuid.UUID), args.Error(2)
}

func (m *mockNotificationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	args := m.Called(ctx, batchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *mockNotificationService) Cancel(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) MarkAsProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error {
	args := m.Called(ctx, id, providerMsgID)
	return args.Error(0)
}

func (m *mockNotificationService) MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error {
	args := m.Called(ctx, id, reason, retryCount)
	return args.Error(0)
}

func (m *mockNotificationService) MarkAsQueued(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNotificationService) MarkAsRetrying(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Processing Service Mock ---

type mockNotificationProcessingService struct {
	mock.Mock
}

func (m *mockNotificationProcessingService) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	args := m.Called(ctx, req, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationProcessingService) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(uuid.UUID), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(uuid.UUID), args.Error(2)
}

func (m *mockNotificationProcessingService) ProcessAndSend(ctx context.Context, id uuid.UUID) (*domain.Notification, bool, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.Notification), args.Bool(1), args.Error(2)
}

func (m *mockNotificationProcessingService) HandleDeliveryFailure(ctx context.Context, id uuid.UUID, retryCount int, maxRetries int) (*domain.Notification, error) {
	args := m.Called(ctx, id, retryCount, maxRetries)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationProcessingService) RecoverStuckNotifications(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockNotificationProcessingService) PublishDueScheduled(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Helpers ---

func setupTestApp(svc *mockNotificationService, processingSvc *mockNotificationProcessingService) *fiber.App {
	app := fiber.New()
	handler := NewNotificationController(svc, processingSvc)
	handler.RegisterRoutes(app.Group("/api/v1"))
	return app
}

func parseAPIResponse(t *testing.T, body io.Reader) response.APIResponse {
	t.Helper()
	var resp response.APIResponse
	err := json.NewDecoder(body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// --- Create Tests ---

func TestController_Create_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()

	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), (*string)(nil)).
		Return(&domain.Notification{
			ID:        notifID,
			Recipient: "+1234567890",
			Channel:   domain.NotificationChannelSMS,
			Content:   "Hello World",
			Priority:  domain.NotificationPriorityNormal,
			Status:    domain.NotificationStatusQueued,
		}, nil)

	body := `{"recipient":"+1234567890","channel":"sms","content":"Hello World"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var notifResp domain.NotificationResponse
	err = json.Unmarshal(data, &notifResp)
	require.NoError(t, err)

	assert.Equal(t, notifID, notifResp.ID)
	assert.Equal(t, "queued", notifResp.Status)

	processingSvc.AssertExpectations(t)
}

func TestController_Create_WithIdempotencyKey(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	key := "my-unique-key"

	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), &key).
		Return(&domain.Notification{
			ID:       notifID,
			Status:   domain.NotificationStatusQueued,
			Channel:  domain.NotificationChannelSMS,
			Priority: domain.NotificationPriorityNormal,
		}, nil)

	body := `{"recipient":"+1234567890","channel":"sms","content":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", key)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	processingSvc.AssertExpectations(t)
}

func TestController_Create_Scheduled_NoPublish(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	futureTime := time.Now().UTC().Add(24 * time.Hour)
	notifID := uuid.New()

	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), (*string)(nil)).
		Return(&domain.Notification{
			ID:          notifID,
			Status:      domain.NotificationStatusScheduled,
			Channel:     domain.NotificationChannelEmail,
			Priority:    domain.NotificationPriorityNormal,
			ScheduledAt: &futureTime,
		}, nil)

	body := fmt.Sprintf(`{"recipient":"user@example.com","channel":"email","content":"Scheduled msg","scheduledAt":"%s"}`, futureTime.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var notifResp domain.NotificationResponse
	err = json.Unmarshal(data, &notifResp)
	require.NoError(t, err)
	assert.Equal(t, "scheduled", notifResp.Status)

	processingSvc.AssertExpectations(t)
}

func TestController_Create_InvalidBody(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_BODY", apiResp.Error.Code)
}

func TestController_Create_ValidationError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	// Missing required fields (no recipient, no channel, no content/templateId).
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

func TestController_Create_DuplicateIdempotencyKey(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	key := "duplicate-key"
	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), &key).
		Return(nil, domain.ErrNotificationDuplicateIdempotencyKey)

	body := `{"recipient":"+1234567890","channel":"sms","content":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", key)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "DUPLICATE_IDEMPOTENCY_KEY", apiResp.Error.Code)

	processingSvc.AssertExpectations(t)
}

func TestController_Create_PublishFails_StillReturns201(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), (*string)(nil)).
		Return(&domain.Notification{
			ID:       notifID,
			Status:   domain.NotificationStatusPending,
			Channel:  domain.NotificationChannelSMS,
			Priority: domain.NotificationPriorityNormal,
		}, nil)

	body := `{"recipient":"+1234567890","channel":"sms","content":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	// Status remains pending when publish fails.
	data, _ := json.Marshal(apiResp.Data)
	var notifResp domain.NotificationResponse
	err = json.Unmarshal(data, &notifResp)
	require.NoError(t, err)
	assert.Equal(t, "pending", notifResp.Status)

	processingSvc.AssertExpectations(t)
}

func TestController_Create_GenericError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), (*string)(nil)).
		Return(nil, fmt.Errorf("unexpected db failure"))

	body := `{"recipient":"+1234567890","channel":"sms","content":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	processingSvc.AssertExpectations(t)
}

// --- CreateBatch Tests ---

func TestController_CreateBatch_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	batchID := uuid.New()
	notifID1 := uuid.New()
	notifID2 := uuid.New()

	notifications := []*domain.Notification{
		{ID: notifID1, Status: domain.NotificationStatusQueued, Channel: domain.NotificationChannelSMS, Priority: domain.NotificationPriorityNormal, BatchID: &batchID},
		{ID: notifID2, Status: domain.NotificationStatusQueued, Channel: domain.NotificationChannelSMS, Priority: domain.NotificationPriorityNormal, BatchID: &batchID},
	}

	processingSvc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.NotificationBatchCreateRequest")).
		Return(notifications, batchID, nil)

	body := `{"notifications":[{"recipient":"+905551111111","channel":"sms","content":"Hello 1"},{"recipient":"+905552222222","channel":"sms","content":"Hello 2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	processingSvc.AssertExpectations(t)
}

func TestController_CreateBatch_InvalidBody(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestController_CreateBatch_ValidationError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	// Empty notifications array.
	body := `{"notifications":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

func TestController_CreateBatch_MixedScheduledAndPending(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	batchID := uuid.New()
	notifID1 := uuid.New()
	notifID2 := uuid.New()
	futureTime := time.Now().UTC().Add(24 * time.Hour)

	notifications := []*domain.Notification{
		{ID: notifID1, Status: domain.NotificationStatusQueued, Channel: domain.NotificationChannelSMS, Priority: domain.NotificationPriorityNormal, BatchID: &batchID},
		{ID: notifID2, Status: domain.NotificationStatusScheduled, Channel: domain.NotificationChannelSMS, Priority: domain.NotificationPriorityNormal, BatchID: &batchID, ScheduledAt: &futureTime},
	}

	processingSvc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.NotificationBatchCreateRequest")).
		Return(notifications, batchID, nil)

	body := fmt.Sprintf(`{"notifications":[{"recipient":"+905551111111","channel":"sms","content":"Hello 1"},{"recipient":"+905552222222","channel":"sms","content":"Scheduled","scheduledAt":"%s"}]}`, futureTime.Format(time.RFC3339Nano))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	processingSvc.AssertExpectations(t)
}

func TestController_CreateBatch_AppError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	processingSvc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.NotificationBatchCreateRequest")).
		Return(nil, uuid.Nil, errs.NewAppError("BATCH_TOO_LARGE", "batch size exceeds maximum of 1000", http.StatusBadRequest))

	body := `{"notifications":[{"recipient":"+905551111111","channel":"sms","content":"Hello 1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "BATCH_TOO_LARGE", apiResp.Error.Code)

	processingSvc.AssertExpectations(t)
}

func TestController_CreateBatch_GenericError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	processingSvc.On("CreateBatch", mock.Anything, mock.AnythingOfType("domain.NotificationBatchCreateRequest")).
		Return(nil, uuid.Nil, fmt.Errorf("db connection lost"))

	body := `{"notifications":[{"recipient":"+905551111111","channel":"sms","content":"Hello 1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	processingSvc.AssertExpectations(t)
}

// --- GetByID Tests ---

func TestController_GetByID_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("GetByID", mock.Anything, notifID).Return(&domain.Notification{
		ID:        notifID,
		Recipient: "+1234567890",
		Channel:   domain.NotificationChannelSMS,
		Status:    domain.NotificationStatusPending,
		Priority:  domain.NotificationPriorityNormal,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+notifID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	svc.AssertExpectations(t)
}

func TestController_GetByID_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/not-a-uuid", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_GetByID_NotFound(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("GetByID", mock.Anything, notifID).Return(nil, domain.ErrNotificationNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+notifID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "NOTIFICATION_NOT_FOUND", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_GetByID_InternalError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("GetByID", mock.Anything, notifID).Return(nil, fmt.Errorf("db connection failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+notifID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// --- GetByBatchID Tests ---

func TestController_GetByBatchID_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	batchID := uuid.New()
	notifications := []*domain.Notification{
		{ID: uuid.New(), BatchID: &batchID, Channel: domain.NotificationChannelSMS, Status: domain.NotificationStatusPending, Priority: domain.NotificationPriorityNormal},
		{ID: uuid.New(), BatchID: &batchID, Channel: domain.NotificationChannelSMS, Status: domain.NotificationStatusPending, Priority: domain.NotificationPriorityNormal},
	}

	svc.On("GetByBatchID", mock.Anything, batchID).Return(notifications, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/batch/"+batchID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	svc.AssertExpectations(t)
}

func TestController_GetByBatchID_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/batch/invalid", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_GetByBatchID_AppError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	batchID := uuid.New()
	svc.On("GetByBatchID", mock.Anything, batchID).
		Return(nil, errs.NewAppError("BATCH_NOT_FOUND", "batch not found", http.StatusNotFound))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/batch/"+batchID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "BATCH_NOT_FOUND", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_GetByBatchID_GenericError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	batchID := uuid.New()
	svc.On("GetByBatchID", mock.Anything, batchID).
		Return(nil, fmt.Errorf("database timeout"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/batch/"+batchID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// --- List Tests ---

func TestController_List_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	expectedFilter := domain.NotificationListFilter{
		Limit:  20,
		Offset: 0,
	}

	notifications := []*domain.Notification{
		{ID: uuid.New(), Channel: domain.NotificationChannelSMS, Status: domain.NotificationStatusPending, Priority: domain.NotificationPriorityNormal},
	}

	svc.On("List", mock.Anything, expectedFilter).Return(notifications, int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	svc.AssertExpectations(t)
}

func TestController_List_WithFilters(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	expectedFilter := domain.NotificationListFilter{
		Status:  "pending",
		Channel: "sms",
		Limit:   10,
		Offset:  5,
	}

	svc.On("List", mock.Anything, expectedFilter).Return([]*domain.Notification{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?status=pending&channel=sms&limit=10&offset=5", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	svc.AssertExpectations(t)
}

func TestController_List_WithDateFilters(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	startDate, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	endDate, _ := time.Parse(time.RFC3339, "2026-12-31T23:59:59Z")

	expectedFilter := domain.NotificationListFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     20,
		Offset:    0,
	}

	svc.On("List", mock.Anything, expectedFilter).Return([]*domain.Notification{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?startDate=2026-01-01T00:00:00Z&endDate=2026-12-31T23:59:59Z", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	svc.AssertExpectations(t)
}

func TestController_List_InvalidStartDate(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?startDate=not-a-date", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_PARAM", apiResp.Error.Code)
}

func TestController_List_InvalidEndDate(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?endDate=not-a-date", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_PARAM", apiResp.Error.Code)
}

func TestController_List_LimitCapped(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	// Limit > 100 should be capped to 100 by Normalize().
	expectedFilter := domain.NotificationListFilter{
		Limit:  100,
		Offset: 0,
	}

	svc.On("List", mock.Anything, expectedFilter).Return([]*domain.Notification{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=500", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	svc.AssertExpectations(t)
}

func TestController_List_AppError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	expectedFilter := domain.NotificationListFilter{
		Limit:  20,
		Offset: 0,
	}

	svc.On("List", mock.Anything, expectedFilter).
		Return(nil, int64(0), errs.NewAppError("INVALID_STATUS", "invalid notification status", http.StatusBadRequest))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_STATUS", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_List_GenericError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	expectedFilter := domain.NotificationListFilter{
		Limit:  20,
		Offset: 0,
	}

	svc.On("List", mock.Anything, expectedFilter).
		Return(nil, int64(0), fmt.Errorf("query execution failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// --- Cancel Tests ---

func TestController_Cancel_Success(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("Cancel", mock.Anything, notifID).Return(&domain.Notification{
		ID:       notifID,
		Status:   domain.NotificationStatusCancelled,
		Channel:  domain.NotificationChannelSMS,
		Priority: domain.NotificationPriorityNormal,
	}, nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/cancel", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var notifResp domain.NotificationResponse
	err = json.Unmarshal(data, &notifResp)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", notifResp.Status)

	svc.AssertExpectations(t)
}

func TestController_Cancel_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/invalid/cancel", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_Cancel_AlreadySent(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("Cancel", mock.Anything, notifID).Return(nil, domain.ErrNotificationAlreadySent)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/cancel", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "NOTIFICATION_ALREADY_SENT", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_Cancel_NotFound(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("Cancel", mock.Anything, notifID).Return(nil, domain.ErrNotificationNotFound)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/cancel", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	svc.AssertExpectations(t)
}

func TestController_Cancel_GenericError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	notifID := uuid.New()
	svc.On("Cancel", mock.Anything, notifID).Return(nil, fmt.Errorf("unexpected failure"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/cancel", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// --- Service CreateFailed Tests ---

func TestController_Create_ServiceError(t *testing.T) {
	svc := new(mockNotificationService)
	processingSvc := new(mockNotificationProcessingService)
	app := setupTestApp(svc, processingSvc)

	processingSvc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationCreateRequest"), (*string)(nil)).
		Return(nil, errs.NewAppError("TEMPLATE_NOT_FOUND", "template not found", http.StatusNotFound))

	body := `{"recipient":"+1234567890","channel":"sms","templateId":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NOT_FOUND", apiResp.Error.Code)

	processingSvc.AssertExpectations(t)
}
