package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationTemplate struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Channel   string         `gorm:"type:varchar(10);not null" json:"channel"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationTemplate) TableName() string { return "templates" }
