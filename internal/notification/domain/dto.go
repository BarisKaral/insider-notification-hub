package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationCreateRequest represents the payload for creating a single notification.
// Either Content or (TemplateID + Variables) must be provided, but not both.
type NotificationCreateRequest struct {
	Recipient   string            `json:"recipient" validate:"required"`
	Channel     string            `json:"channel" validate:"required,oneof=sms email push"`
	Content     *string           `json:"content,omitempty"`
	Priority    string            `json:"priority" validate:"omitempty,oneof=high normal low"`
	ScheduledAt *time.Time        `json:"scheduledAt,omitempty"`
	TemplateID  *uuid.UUID        `json:"templateId,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

// NotificationBatchCreateRequest represents the payload for creating multiple notifications at once.
type NotificationBatchCreateRequest struct {
	Notifications []NotificationCreateRequest `json:"notifications" validate:"required,min=1,max=1000"`
}

// NotificationListFilter holds query parameters for listing notifications.
type NotificationListFilter struct {
	Status    string
	Channel   string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int // default 20, max 100
	Offset    int // default 0
}

// NotificationResponse is the API representation of a Notification.
type NotificationResponse struct {
	ID            uuid.UUID  `json:"id"`
	Recipient     string     `json:"recipient"`
	Channel       string     `json:"channel"`
	Content       string     `json:"content"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	BatchID       *uuid.UUID `json:"batchId,omitempty"`
	TemplateID    *uuid.UUID `json:"templateId,omitempty"`
	ProviderMsgID *string    `json:"providerMessageId,omitempty"`
	RetryCount    int        `json:"retryCount"`
	ScheduledAt   *time.Time `json:"scheduledAt,omitempty"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
	FailedAt      *time.Time `json:"failedAt,omitempty"`
	FailureReason *string    `json:"failureReason,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// NotificationPaginatedResponse wraps a list of notifications with pagination metadata.
type NotificationPaginatedResponse struct {
	Items  []NotificationResponse `json:"items"`
	Total  int64                  `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// NotificationBatchResponse wraps a batch of notifications with their shared batch ID.
type NotificationBatchResponse struct {
	BatchID       uuid.UUID              `json:"batchId"`
	Notifications []NotificationResponse `json:"notifications"`
}

// ToNotificationResponse maps a Notification entity to a NotificationResponse DTO.
func ToNotificationResponse(n *Notification) NotificationResponse {
	return NotificationResponse{
		ID:            n.ID,
		Recipient:     n.Recipient,
		Channel:       string(n.Channel),
		Content:       n.Content,
		Priority:      string(n.Priority),
		Status:        string(n.Status),
		BatchID:       n.BatchID,
		TemplateID:    n.TemplateID,
		ProviderMsgID: n.ProviderMsgID,
		RetryCount:    n.RetryCount,
		ScheduledAt:   n.ScheduledAt,
		SentAt:        n.SentAt,
		FailedAt:      n.FailedAt,
		FailureReason: n.FailureReason,
		CreatedAt:     n.CreatedAt,
	}
}

// ToNotificationResponseList maps a slice of Notification entities to a slice of NotificationResponse DTOs.
func ToNotificationResponseList(notifications []*Notification) []NotificationResponse {
	responses := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = ToNotificationResponse(n)
	}
	return responses
}
