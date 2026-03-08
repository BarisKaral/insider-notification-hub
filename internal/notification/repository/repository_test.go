package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/baris/notification-hub/internal/notification/domain"
)

func TestRepositoryInterfaceCompliance(t *testing.T) {
	// Compile-time check: repository implements NotificationRepository
	var _ NotificationRepository = (*repository)(nil)
}

func TestIsUniqueViolation(t *testing.T) {
	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, isUniqueViolation(nil))
	})

	t.Run("returns false for non-pg error", func(t *testing.T) {
		assert.False(t, isUniqueViolation(assert.AnError))
	})
}

func TestToResponse_MapsAllFields(t *testing.T) {
	now := time.Now().UTC()
	batchID := uuid.New()
	templateID := uuid.New()
	providerID := "provider-123"
	failReason := "timeout"

	n := &domain.Notification{
		ID:            uuid.New(),
		Recipient:     "+905551234567",
		Channel:       domain.NotificationChannelSMS,
		Content:       "Hello",
		Priority:      domain.NotificationPriorityHigh,
		Status:        domain.NotificationStatusSent,
		BatchID:       &batchID,
		TemplateID:    &templateID,
		ProviderMsgID: &providerID,
		RetryCount:    2,
		ScheduledAt:   &now,
		SentAt:        &now,
		FailedAt:      &now,
		FailureReason: &failReason,
		CreatedAt:     now,
	}

	resp := domain.ToNotificationResponse(n)

	assert.Equal(t, n.ID, resp.ID)
	assert.Equal(t, n.Recipient, resp.Recipient)
	assert.Equal(t, string(domain.NotificationChannelSMS), resp.Channel)
	assert.Equal(t, n.Content, resp.Content)
	assert.Equal(t, string(domain.NotificationPriorityHigh), resp.Priority)
	assert.Equal(t, string(domain.NotificationStatusSent), resp.Status)
	assert.Equal(t, &batchID, resp.BatchID)
	assert.Equal(t, &templateID, resp.TemplateID)
	assert.Equal(t, &providerID, resp.ProviderMsgID)
	assert.Equal(t, 2, resp.RetryCount)
	assert.Equal(t, &now, resp.ScheduledAt)
	assert.Equal(t, &now, resp.SentAt)
	assert.Equal(t, &now, resp.FailedAt)
	assert.Equal(t, &failReason, resp.FailureReason)
	assert.Equal(t, now, resp.CreatedAt)
}

func TestToResponseList(t *testing.T) {
	n1 := &domain.Notification{ID: uuid.New(), Recipient: "a", Channel: domain.NotificationChannelSMS, Status: domain.NotificationStatusPending}
	n2 := &domain.Notification{ID: uuid.New(), Recipient: "b", Channel: domain.NotificationChannelEmail, Status: domain.NotificationStatusSent}

	responses := domain.ToNotificationResponseList([]*domain.Notification{n1, n2})

	assert.Len(t, responses, 2)
	assert.Equal(t, n1.ID, responses[0].ID)
	assert.Equal(t, n2.ID, responses[1].ID)
}

func TestToResponseList_Empty(t *testing.T) {
	responses := domain.ToNotificationResponseList([]*domain.Notification{})
	assert.Empty(t, responses)
	assert.NotNil(t, responses)
}
