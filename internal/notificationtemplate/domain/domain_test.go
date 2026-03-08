package domain

import (
	"net/http"
	"testing"
	"time"

	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationTemplate_TableName(t *testing.T) {
	tpl := NotificationTemplate{}
	assert.Equal(t, "templates", tpl.TableName())
}

func TestToNotificationTemplateResponse(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	tpl := &NotificationTemplate{
		ID:        id,
		Name:      "test",
		Channel:   "sms",
		Content:   "hello {{name}}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := ToNotificationTemplateResponse(tpl)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, "test", resp.Name)
	assert.Equal(t, "sms", resp.Channel)
	assert.Equal(t, "hello {{name}}", resp.Content)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestToNotificationTemplateResponseList(t *testing.T) {
	templates := []*NotificationTemplate{
		{ID: uuid.New(), Name: "a", Channel: "sms", Content: "c1"},
		{ID: uuid.New(), Name: "b", Channel: "email", Content: "c2"},
	}

	responses := ToNotificationTemplateResponseList(templates)
	assert.Len(t, responses, 2)
	assert.Equal(t, "a", responses[0].Name)
	assert.Equal(t, "b", responses[1].Name)
}

func TestToNotificationTemplateResponseList_Empty(t *testing.T) {
	responses := ToNotificationTemplateResponseList([]*NotificationTemplate{})
	assert.Len(t, responses, 0)
}

func TestNotificationTemplateErrors(t *testing.T) {
	assert.Equal(t, 404, ErrNotificationTemplateNotFound.StatusCode)
	assert.Equal(t, 409, ErrNotificationTemplateNameExists.StatusCode)
	assert.Equal(t, 500, ErrNotificationTemplateCreateFailed.StatusCode)
}

// --- CreateRequest Validate tests ---

func TestCreateRequest_Validate_Valid(t *testing.T) {
	req := NotificationTemplateCreateRequest{
		Name:    "order_shipped",
		Channel: "sms",
		Content: "Your order has been shipped.",
	}
	err := req.Validate()
	assert.NoError(t, err)
}

func TestCreateRequest_Validate_MissingName(t *testing.T) {
	req := NotificationTemplateCreateRequest{
		Channel: "sms",
		Content: "hello",
	}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestCreateRequest_Validate_MissingChannel(t *testing.T) {
	req := NotificationTemplateCreateRequest{
		Name:    "test",
		Content: "hello",
	}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestCreateRequest_Validate_InvalidChannel(t *testing.T) {
	req := NotificationTemplateCreateRequest{
		Name:    "test",
		Channel: "fax",
		Content: "hello",
	}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestCreateRequest_Validate_MissingContent(t *testing.T) {
	req := NotificationTemplateCreateRequest{
		Name:    "test",
		Channel: "sms",
	}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestCreateRequest_Validate_AllValidChannels(t *testing.T) {
	channels := []string{"sms", "email", "push"}

	for _, ch := range channels {
		t.Run(ch, func(t *testing.T) {
			req := NotificationTemplateCreateRequest{
				Name:    "test_" + ch,
				Channel: ch,
				Content: "hello from " + ch,
			}
			err := req.Validate()
			assert.NoError(t, err, "channel %q should be valid", ch)
		})
	}
}

// --- UpdateRequest Validate tests ---

func TestUpdateRequest_Validate_ValidWithNameOnly(t *testing.T) {
	name := "new_name"
	req := NotificationTemplateUpdateRequest{Name: &name}
	err := req.Validate()
	assert.NoError(t, err)
}

func TestUpdateRequest_Validate_ValidWithChannelOnly(t *testing.T) {
	channel := "email"
	req := NotificationTemplateUpdateRequest{Channel: &channel}
	err := req.Validate()
	assert.NoError(t, err)
}

func TestUpdateRequest_Validate_ValidWithContentOnly(t *testing.T) {
	content := "new content"
	req := NotificationTemplateUpdateRequest{Content: &content}
	err := req.Validate()
	assert.NoError(t, err)
}

func TestUpdateRequest_Validate_ValidWithAllFields(t *testing.T) {
	name := "updated_name"
	channel := "push"
	content := "updated content"
	req := NotificationTemplateUpdateRequest{
		Name:    &name,
		Channel: &channel,
		Content: &content,
	}
	err := req.Validate()
	assert.NoError(t, err)
}

func TestUpdateRequest_Validate_NoFields(t *testing.T) {
	req := NotificationTemplateUpdateRequest{}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
	assert.Contains(t, appErr.Message, "at least one field must be provided")
}

func TestUpdateRequest_Validate_InvalidChannel(t *testing.T) {
	invalidCh := "telegram"
	req := NotificationTemplateUpdateRequest{Channel: &invalidCh}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestUpdateRequest_Validate_AllValidChannels(t *testing.T) {
	channels := []string{"sms", "email", "push"}

	for _, ch := range channels {
		t.Run(ch, func(t *testing.T) {
			channel := ch
			req := NotificationTemplateUpdateRequest{Channel: &channel}
			err := req.Validate()
			assert.NoError(t, err, "channel %q should be valid", ch)
		})
	}
}

// --- Error type assertion tests ---

func TestCreateRequest_Validate_ReturnsAppError(t *testing.T) {
	req := NotificationTemplateCreateRequest{}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestUpdateRequest_Validate_ReturnsAppError_InvalidChannel(t *testing.T) {
	invalidCh := "fax"
	req := NotificationTemplateUpdateRequest{Channel: &invalidCh}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}

func TestUpdateRequest_Validate_ReturnsAppError_NoFields(t *testing.T) {
	req := NotificationTemplateUpdateRequest{}
	err := req.Validate()
	require.Error(t, err)

	var appErr *errs.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.StatusCode)
}
