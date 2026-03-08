package template

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*Template, error)
	List(ctx context.Context, limit, offset int) ([]*Template, int64, error)
	Update(ctx context.Context, t *Template) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

var _ TemplateRepository = (*repository)(nil)

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, t *Template) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrTemplateNameExists.WithError(err)
		}
		return ErrTemplateCreateFailed.WithError(err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	var t Template
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, ErrTemplateNotFound.WithError(err)
	}
	return &t, nil
}

func (r *repository) List(ctx context.Context, limit, offset int) ([]*Template, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&Template{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var templates []*Template
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

func (r *repository) Update(ctx context.Context, t *Template) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Template{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
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
