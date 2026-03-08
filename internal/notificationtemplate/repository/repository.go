package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
)

type NotificationTemplateRepository interface {
	Create(ctx context.Context, t *domain.NotificationTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error)
	List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error)
	Update(ctx context.Context, t *domain.NotificationTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	database *gorm.DB
}

var _ NotificationTemplateRepository = (*repository)(nil)

func NewNotificationTemplateRepository(database *gorm.DB) *repository {
	return &repository{database: database}
}

func (r *repository) Create(ctx context.Context, t *domain.NotificationTemplate) error {
	if err := r.database.WithContext(ctx).Create(t).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrNotificationTemplateNameExists.WithError(err)
		}
		return domain.ErrNotificationTemplateCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.NotificationTemplate, error) {
	var template domain.NotificationTemplate
	if err := r.database.WithContext(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotificationTemplateNotFound
		}
		return nil, domain.ErrNotificationTemplateNotFound.WithError(err)
	}
	return &template, nil
}

func (r *repository) List(ctx context.Context, limit, offset int) ([]*domain.NotificationTemplate, int64, error) {
	var total int64
	if err := r.database.WithContext(ctx).Model(&domain.NotificationTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var templates []*domain.NotificationTemplate
	if err := r.database.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

func (r *repository) Update(ctx context.Context, t *domain.NotificationTemplate) error {
	return r.database.WithContext(ctx).Save(t).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.database.WithContext(ctx).Where("id = ?", id).Delete(&domain.NotificationTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotificationTemplateNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
