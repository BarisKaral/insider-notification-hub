package notificationtemplate

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type NotificationTemplateRepository interface {
	Create(ctx context.Context, t *NotificationTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error)
	List(ctx context.Context, limit, offset int) ([]*NotificationTemplate, int64, error)
	Update(ctx context.Context, t *NotificationTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

var _ NotificationTemplateRepository = (*repository)(nil)

func NewNotificationTemplateRepository(db *gorm.DB) NotificationTemplateRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, t *NotificationTemplate) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrNotificationTemplateNameExists.WithError(err)
		}
		return ErrNotificationTemplateCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error) {
	var t NotificationTemplate
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationTemplateNotFound
		}
		return nil, ErrNotificationTemplateNotFound.WithError(err)
	}
	return &t, nil
}

func (r *repository) List(ctx context.Context, limit, offset int) ([]*NotificationTemplate, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&NotificationTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var templates []*NotificationTemplate
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

func (r *repository) Update(ctx context.Context, t *NotificationTemplate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&NotificationTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotificationTemplateNotFound
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
