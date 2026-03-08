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

	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/bariskaral/insider-notification-hub/pkg/response"
)

// --- Service Mock ---

type mockNotificationTemplateService struct {
	mock.Mock
}

func (m *mockNotificationTemplateService) Create(ctx context.Context, req domain.NotificationTemplateCreateRequest) (*domain.NotificationTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.NotificationTemplate), args.Error(1)
}

func (m *mockNotificationTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.NotificationTemplate), args.Error(1)
}

func (m *mockNotificationTemplateService) List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.NotificationTemplate), args.Get(1).(int64), args.Error(2)
}

func (m *mockNotificationTemplateService) Update(ctx context.Context, id uuid.UUID, req domain.NotificationTemplateUpdateRequest) (*domain.NotificationTemplate, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.NotificationTemplate), args.Error(1)
}

func (m *mockNotificationTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNotificationTemplateService) Render(ctx context.Context, templateID uuid.UUID, variables map[string]string) (string, error) {
	args := m.Called(ctx, templateID, variables)
	return args.String(0), args.Error(1)
}

// --- Helpers ---

func setupTestApp(svc *mockNotificationTemplateService) *fiber.App {
	app := fiber.New()
	ctrl := NewNotificationTemplateController(svc)
	ctrl.RegisterRoutes(app.Group("/api/v1"))
	return app
}

func parseAPIResponse(t *testing.T, body io.Reader) response.APIResponse {
	t.Helper()
	var resp response.APIResponse
	err := json.NewDecoder(body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

func sampleTemplate() *domain.NotificationTemplate {
	return &domain.NotificationTemplate{
		ID:        uuid.New(),
		Name:      "order_shipped",
		Channel:   "sms",
		Content:   "Your order {{orderId}} has been shipped.",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// ===================== Create Tests =====================

func TestController_Create_Success(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	tmpl := sampleTemplate()

	svc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationTemplateCreateRequest")).
		Return(tmpl, nil)

	body := `{"name":"order_shipped","channel":"sms","content":"Your order {{orderId}} has been shipped."}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var tmplResp domain.NotificationTemplateResponse
	err = json.Unmarshal(data, &tmplResp)
	require.NoError(t, err)

	assert.Equal(t, tmpl.ID, tmplResp.ID)
	assert.Equal(t, "order_shipped", tmplResp.Name)
	assert.Equal(t, "sms", tmplResp.Channel)

	svc.AssertExpectations(t)
}

func TestController_Create_InvalidBody(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_BODY", apiResp.Error.Code)
}

func TestController_Create_ValidationError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	// Missing required fields triggers AppError from Validate()
	body := `{"name":"","channel":"","content":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

func TestController_Create_ServiceAppError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationTemplateCreateRequest")).
		Return(nil, domain.ErrNotificationTemplateNameExists)

	body := `{"name":"existing","channel":"sms","content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NAME_EXISTS", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_Create_ServiceGenericError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("Create", mock.Anything, mock.AnythingOfType("domain.NotificationTemplateCreateRequest")).
		Return(nil, fmt.Errorf("db connection lost"))

	body := `{"name":"test","channel":"sms","content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== GetByID Tests =====================

func TestController_GetByID_Success(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	tmpl := sampleTemplate()

	svc.On("GetByID", mock.Anything, tmpl.ID).Return(tmpl, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/"+tmpl.ID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var tmplResp domain.NotificationTemplateResponse
	err = json.Unmarshal(data, &tmplResp)
	require.NoError(t, err)

	assert.Equal(t, tmpl.ID, tmplResp.ID)
	assert.Equal(t, tmpl.Name, tmplResp.Name)

	svc.AssertExpectations(t)
}

func TestController_GetByID_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/not-a-uuid", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_GetByID_NotFound(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotificationTemplateNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NOT_FOUND", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_GetByID_InternalError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("GetByID", mock.Anything, id).Return(nil, fmt.Errorf("unexpected failure"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== List Tests =====================

func TestController_List_Success_DefaultParams(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	templates := []*domain.NotificationTemplate{sampleTemplate()}

	svc.On("List", mock.Anything, 20, 0).Return(templates, int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var listResp map[string]interface{}
	err = json.Unmarshal(data, &listResp)
	require.NoError(t, err)

	assert.Equal(t, float64(1), listResp["total"])
	assert.Equal(t, float64(20), listResp["limit"])
	assert.Equal(t, float64(0), listResp["offset"])

	items, ok := listResp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)

	svc.AssertExpectations(t)
}

func TestController_List_CustomLimitAndOffset(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 10, 5).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/?limit=10&offset=5", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var listResp map[string]interface{}
	err = json.Unmarshal(data, &listResp)
	require.NoError(t, err)

	assert.Equal(t, float64(10), listResp["limit"])
	assert.Equal(t, float64(5), listResp["offset"])

	svc.AssertExpectations(t)
}

func TestController_List_LimitCappedAt100(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 100, 0).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/?limit=500", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var listResp map[string]interface{}
	err = json.Unmarshal(data, &listResp)
	require.NoError(t, err)

	assert.Equal(t, float64(100), listResp["limit"])

	svc.AssertExpectations(t)
}

func TestController_List_NegativeOffsetDefaultsToZero(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 20, 0).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/?offset=-5", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var listResp map[string]interface{}
	err = json.Unmarshal(data, &listResp)
	require.NoError(t, err)

	assert.Equal(t, float64(0), listResp["offset"])

	svc.AssertExpectations(t)
}

func TestController_List_NegativeLimitDefaultsTo20(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 20, 0).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/?limit=-10", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	data, _ := json.Marshal(apiResp.Data)
	var listResp map[string]interface{}
	err = json.Unmarshal(data, &listResp)
	require.NoError(t, err)

	assert.Equal(t, float64(20), listResp["limit"])

	svc.AssertExpectations(t)
}

func TestController_List_ServiceError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 20, 0).Return(nil, int64(0), fmt.Errorf("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== Update Tests =====================

func TestController_Update_Success(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	tmpl := sampleTemplate()
	newName := "updated_name"

	svc.On("Update", mock.Anything, tmpl.ID, mock.AnythingOfType("domain.NotificationTemplateUpdateRequest")).
		Return(tmpl, nil)

	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+tmpl.ID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.True(t, apiResp.Success)

	svc.AssertExpectations(t)
}

func TestController_Update_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_Update_InvalidBody(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_BODY", apiResp.Error.Code)
}

func TestController_Update_ValidationError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	// Empty body -> no fields provided -> validation error
	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

func TestController_Update_NotFound(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()

	svc.On("Update", mock.Anything, id, mock.AnythingOfType("domain.NotificationTemplateUpdateRequest")).
		Return(nil, domain.ErrNotificationTemplateNotFound)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NOT_FOUND", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_Update_InternalError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()

	svc.On("Update", mock.Anything, id, mock.AnythingOfType("domain.NotificationTemplateUpdateRequest")).
		Return(nil, fmt.Errorf("db error"))

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_Update_ValidationError_InvalidChannel(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	body := `{"channel":"fax"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

// ===================== Delete Tests =====================

func TestController_Delete_Success(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	svc.AssertExpectations(t)
}

func TestController_Delete_InvalidUUID(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification-templates/not-a-uuid", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INVALID_ID", apiResp.Error.Code)
}

func TestController_Delete_NotFound(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("Delete", mock.Anything, id).Return(domain.ErrNotificationTemplateNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NOT_FOUND", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

func TestController_Delete_InternalError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("Delete", mock.Anything, id).Return(fmt.Errorf("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== Interface Compliance =====================

func TestController_ImplementsInterface(t *testing.T) {
	var _ NotificationTemplateController = (*controller)(nil)
}

// ===================== RegisterRoutes Test =====================

func TestController_RegisterRoutes(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := fiber.New()
	ctrl := NewNotificationTemplateController(svc)
	ctrl.RegisterRoutes(app.Group("/api/v1"))

	// Verify routes are registered by checking that known endpoints return
	// something other than 404 "Cannot ..." (Fiber's default for unregistered routes).
	// A POST to the templates root should at least reach the handler (400 for missing body).
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	// Even with an empty body, the handler will return 400 (not 404)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ===================== Create with AppError from validation =====================

func TestController_Create_ValidationReturnsAppError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	// Invalid channel value triggers AppError from Validate()
	body := `{"name":"test","channel":"fax","content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-templates/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
}

// ===================== GetByID with AppError wrapping =====================

func TestController_GetByID_ServiceAppError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	appErr := errs.NewAppError("CUSTOM_ERROR", "custom error", http.StatusForbidden)
	svc.On("GetByID", mock.Anything, id).Return(nil, appErr)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "CUSTOM_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== List with zero limit defaults to 20 =====================

func TestController_List_ZeroLimitDefaultsTo20(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	svc.On("List", mock.Anything, 20, 0).Return([]*domain.NotificationTemplate{}, int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-templates/?limit=0", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	svc.AssertExpectations(t)
}

// ===================== Update with AppError from service =====================

func TestController_Update_ServiceAppError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	svc.On("Update", mock.Anything, id, mock.AnythingOfType("domain.NotificationTemplateUpdateRequest")).
		Return(nil, domain.ErrNotificationTemplateNameExists)

	body := `{"name":"existing_name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-templates/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "TEMPLATE_NAME_EXISTS", apiResp.Error.Code)

	svc.AssertExpectations(t)
}

// ===================== Delete with AppError from service =====================

func TestController_Delete_ServiceAppError(t *testing.T) {
	svc := new(mockNotificationTemplateService)
	app := setupTestApp(svc)

	id := uuid.New()
	appErr := errs.NewAppError("CUSTOM_DELETE_ERROR", "cannot delete", http.StatusForbidden)
	svc.On("Delete", mock.Anything, id).Return(appErr)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notification-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	apiResp := parseAPIResponse(t, resp.Body)
	assert.False(t, apiResp.Success)
	assert.Equal(t, "CUSTOM_DELETE_ERROR", apiResp.Error.Code)

	svc.AssertExpectations(t)
}
