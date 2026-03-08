package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/baris/notification-hub/internal/notification/domain"
)

// NotificationRepository defines the data access interface for notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	CreateBatch(ctx context.Context, notifications []*domain.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error)
	List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error)
	Update(ctx context.Context, n *domain.Notification) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error)
	GetForProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*domain.Notification, error)
	GetDueScheduledNotifications(ctx context.Context) ([]*domain.Notification, error)
}

type repository struct {
	database *gorm.DB
}

var _ NotificationRepository = (*repository)(nil)

func NewNotificationRepository(database *gorm.DB) *repository {
	return &repository{database: database}
}

func (r *repository) Create(ctx context.Context, n *domain.Notification) error {
	if err := r.database.WithContext(ctx).Create(n).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrNotificationDuplicateIdempotencyKey.WithError(err)
		}
		return domain.ErrNotificationCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	if err := r.database.WithContext(ctx).Create(notifications).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrNotificationDuplicateIdempotencyKey.WithError(err)
		}
		return domain.ErrNotificationCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	var notification domain.Notification
	if err := r.database.WithContext(ctx).Where("id = ?", id).First(&notification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, domain.ErrNotificationNotFound.WithError(err)
	}
	return &notification, nil
}

func (r *repository) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	if err := r.database.WithContext(ctx).Where("batch_id = ?", batchID).Find(&notifications).Error; err != nil {
		return nil, domain.ErrNotificationNotFound.WithError(err)
	}
	return notifications, nil
}

func (r *repository) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	query := r.database.WithContext(ctx).Model(&domain.Notification{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Channel != "" {
		query = query.Where("channel = ?", filter.Channel)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var notifications []*domain.Notification
	if err := query.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *repository) Update(ctx context.Context, n *domain.Notification) error {
	return r.database.WithContext(ctx).Save(n).Error
}

func (r *repository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	var notification domain.Notification
	if err := r.database.WithContext(ctx).Where("idempotency_key = ?", key).First(&notification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &notification, nil
}

// GetForProcessing retrieves a notification with a row-level lock (SELECT FOR UPDATE)
// to prevent duplicate processing by concurrent consumers.
func (r *repository) GetForProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	var notification domain.Notification
	if err := r.database.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&notification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, domain.ErrNotificationNotFound.WithError(err)
	}
	return &notification, nil
}

// GetRecoverableNotifications finds notifications stuck in pending state
// beyond the given stale duration.
func (r *repository) GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	cutoff := time.Now().UTC().Add(-staleDuration)
	if err := r.database.WithContext(ctx).
		Where("status = ? AND created_at < ?", domain.NotificationStatusPending, cutoff).
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetDueScheduledNotifications finds scheduled notifications whose scheduled_at time has passed.
func (r *repository) GetDueScheduledNotifications(ctx context.Context) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	if err := r.database.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", domain.NotificationStatusScheduled, time.Now().UTC()).
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
