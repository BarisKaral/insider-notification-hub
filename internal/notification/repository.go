package notification

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NotificationRepository defines the data access interface for notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	CreateBatch(ctx context.Context, notifications []*Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*Notification, error)
	List(ctx context.Context, filter NotificationListFilter) ([]*Notification, int64, error)
	Update(ctx context.Context, n *Notification) error
	GetByIdempotencyKey(ctx context.Context, key string) (*Notification, error)
	GetForProcessing(ctx context.Context, id uuid.UUID) (*Notification, error)
	GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*Notification, error)
	GetDueScheduledNotifications(ctx context.Context) ([]*Notification, error)
}

type repository struct {
	db *gorm.DB
}

var _ NotificationRepository = (*repository)(nil)

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, n *Notification) error {
	if err := r.db.WithContext(ctx).Create(n).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrNotificationDuplicateIdempotencyKey.WithError(err)
		}
		return ErrNotificationCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) CreateBatch(ctx context.Context, notifications []*Notification) error {
	if err := r.db.WithContext(ctx).Create(notifications).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrNotificationDuplicateIdempotencyKey.WithError(err)
		}
		return ErrNotificationCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	var n Notification
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, ErrNotificationNotFound.WithError(err)
	}
	return &n, nil
}

func (r *repository) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*Notification, error) {
	var notifications []*Notification
	if err := r.db.WithContext(ctx).Where("batch_id = ?", batchID).Find(&notifications).Error; err != nil {
		return nil, ErrNotificationNotFound.WithError(err)
	}
	return notifications, nil
}

func (r *repository) List(ctx context.Context, filter NotificationListFilter) ([]*Notification, int64, error) {
	query := r.db.WithContext(ctx).Model(&Notification{})

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

	var notifications []*Notification
	if err := query.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *repository) Update(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Save(n).Error
}

func (r *repository) GetByIdempotencyKey(ctx context.Context, key string) (*Notification, error) {
	var n Notification
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

// GetForProcessing retrieves a notification with a row-level lock (SELECT FOR UPDATE)
// to prevent duplicate processing by concurrent consumers.
func (r *repository) GetForProcessing(ctx context.Context, id uuid.UUID) (*Notification, error) {
	var n Notification
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, ErrNotificationNotFound.WithError(err)
	}
	return &n, nil
}

// GetRecoverableNotifications finds notifications stuck in pending state
// beyond the given stale duration.
func (r *repository) GetRecoverableNotifications(ctx context.Context, staleDuration time.Duration) ([]*Notification, error) {
	var notifications []*Notification
	cutoff := time.Now().UTC().Add(-staleDuration)
	if err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", NotificationStatusPending, cutoff).
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetDueScheduledNotifications finds scheduled notifications whose scheduled_at time has passed.
func (r *repository) GetDueScheduledNotifications(ctx context.Context) ([]*Notification, error) {
	var notifications []*Notification
	if err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", NotificationStatusScheduled, time.Now().UTC()).
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
