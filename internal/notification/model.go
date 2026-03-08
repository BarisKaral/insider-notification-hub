package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Recipient      string             `gorm:"not null;index" json:"recipient"`
	Channel        Channel            `gorm:"type:varchar(10);not null;index" json:"channel"`
	Content        string             `gorm:"type:text;not null" json:"content"`
	Priority       Priority           `gorm:"type:varchar(10);not null;default:'normal';index" json:"priority"`
	Status         NotificationStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	BatchID        *uuid.UUID         `gorm:"type:uuid;index" json:"batchId"`
	IdempotencyKey *string            `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	TemplateID     *uuid.UUID         `gorm:"type:uuid" json:"templateId"`
	TemplateVars   json.RawMessage    `gorm:"type:jsonb" json:"templateVars,omitempty"`
	ProviderMsgID  *string            `gorm:"type:varchar(255)" json:"providerMessageId"`
	RetryCount     int                `gorm:"default:0" json:"retryCount"`
	ScheduledAt    *time.Time         `gorm:"index" json:"scheduledAt"`
	SentAt         *time.Time         `json:"sentAt"`
	FailedAt       *time.Time         `json:"failedAt"`
	FailureReason  *string            `gorm:"type:text" json:"failureReason"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}
