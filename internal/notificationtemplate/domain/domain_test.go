package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
