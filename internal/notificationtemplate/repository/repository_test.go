package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/bariskaral/insider-notification-hub/internal/notificationtemplate/domain"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

// ===================== Interface Compliance =====================

func TestRepositoryInterfaceCompliance(t *testing.T) {
	var _ NotificationTemplateRepository = (*repository)(nil)
}

// ===================== Create Tests =====================

func TestRepository_Create_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	tmpl := &domain.NotificationTemplate{
		ID:      uuid.New(),
		Name:    "order_shipped",
		Channel: "sms",
		Content: "Your order {{orderId}} has been shipped.",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "templates"`).
		WithArgs(
			sqlmock.AnyArg(), // name
			sqlmock.AnyArg(), // channel
			sqlmock.AnyArg(), // content
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at
			sqlmock.AnyArg(), // id
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tmpl.ID))
	mock.ExpectCommit()

	err := repo.Create(ctx, tmpl)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_UniqueViolation(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	tmpl := &domain.NotificationTemplate{
		ID:      uuid.New(),
		Name:    "existing_template",
		Channel: "sms",
		Content: "hello",
	}

	pgErr := &pgconn.PgError{Code: "23505"}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "templates"`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnError(pgErr)
	mock.ExpectRollback()

	err := repo.Create(ctx, tmpl)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TEMPLATE_NAME_EXISTS")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_GenericError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	tmpl := &domain.NotificationTemplate{
		ID:      uuid.New(),
		Name:    "test_template",
		Channel: "sms",
		Content: "hello",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "templates"`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("connection refused"))
	mock.ExpectRollback()

	err := repo.Create(ctx, tmpl)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TEMPLATE_CREATE_FAILED")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetByID Tests =====================

func TestRepository_GetByID_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "name", "channel", "content", "created_at", "updated_at", "deleted_at"}).
		AddRow(id, "test_template", "sms", "hello {{name}}", now, now, nil)

	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE id = .+ AND "templates"\."deleted_at" IS NULL`).
		WithArgs(id, 1).
		WillReturnRows(rows)

	tmpl, err := repo.GetByID(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, id, tmpl.ID)
	assert.Equal(t, "test_template", tmpl.Name)
	assert.Equal(t, "sms", tmpl.Channel)
	assert.Equal(t, "hello {{name}}", tmpl.Content)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE id = .+ AND "templates"\."deleted_at" IS NULL`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	tmpl, err := repo.GetByID(ctx, id)

	assert.Nil(t, tmpl)
	require.Error(t, err)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE id = .+ AND "templates"\."deleted_at" IS NULL`).
		WithArgs(id, 1).
		WillReturnError(errors.New("connection timeout"))

	tmpl, err := repo.GetByID(ctx, id)

	assert.Nil(t, tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TEMPLATE_NOT_FOUND")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== List Tests =====================

func TestRepository_List_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now().UTC()

	// Count query
	mock.ExpectQuery(`SELECT count\(\*\) FROM "templates" WHERE "templates"\."deleted_at" IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Find query - GORM inlines LIMIT and OFFSET values (no placeholders)
	rows := sqlmock.NewRows([]string{"id", "name", "channel", "content", "created_at", "updated_at", "deleted_at"}).
		AddRow(id1, "template_a", "sms", "content a", now, now, nil).
		AddRow(id2, "template_b", "email", "content b", now, now, nil)

	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE "templates"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`).
		WillReturnRows(rows)

	templates, total, err := repo.List(ctx, 20, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, templates, 2)
	assert.Equal(t, "template_a", templates[0].Name)
	assert.Equal(t, "template_b", templates[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_CountError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "templates" WHERE "templates"\."deleted_at" IS NULL`).
		WillReturnError(errors.New("db error"))

	templates, total, err := repo.List(ctx, 20, 0)

	require.Error(t, err)
	assert.Nil(t, templates)
	assert.Equal(t, int64(0), total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_FindError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "templates" WHERE "templates"\."deleted_at" IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE "templates"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`).
		WillReturnError(errors.New("query failed"))

	templates, total, err := repo.List(ctx, 20, 0)

	require.Error(t, err)
	assert.Nil(t, templates)
	assert.Equal(t, int64(0), total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_List_Empty(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "templates" WHERE "templates"\."deleted_at" IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rows := sqlmock.NewRows([]string{"id", "name", "channel", "content", "created_at", "updated_at", "deleted_at"})
	mock.ExpectQuery(`SELECT \* FROM "templates" WHERE "templates"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`).
		WillReturnRows(rows)

	templates, total, err := repo.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, templates)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== Update Tests =====================

func TestRepository_Update_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now().UTC()
	tmpl := &domain.NotificationTemplate{
		ID:        id,
		Name:      "updated_template",
		Channel:   "email",
		Content:   "updated content",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// GORM Save generates: UPDATE "templates" SET "name"=$1,"channel"=$2,"content"=$3,"created_at"=$4,"updated_at"=$5,"deleted_at"=$6
	//   WHERE "templates"."deleted_at" IS NULL AND "id" = $7
	// 7 arguments total
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "templates" SET`).
		WithArgs(
			sqlmock.AnyArg(), // name
			sqlmock.AnyArg(), // channel
			sqlmock.AnyArg(), // content
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at
			sqlmock.AnyArg(), // WHERE id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Update(ctx, tmpl)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_Error(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()
	tmpl := &domain.NotificationTemplate{
		ID:      id,
		Name:    "test",
		Channel: "sms",
		Content: "hello",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "templates" SET`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	err := repo.Update(ctx, tmpl)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== Delete Tests =====================

func TestRepository_Delete_Success(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "templates" SET "deleted_at"=`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Delete(ctx, id)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_NotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "templates" SET "deleted_at"=`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.Delete(ctx, id)

	require.Error(t, err)
	assert.Equal(t, domain.ErrNotificationTemplateNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_DBError(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewNotificationTemplateRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "templates" SET "deleted_at"=`).
		WithArgs(sqlmock.AnyArg(), id).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	err := repo.Delete(ctx, id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== isUniqueViolation Tests =====================

func TestIsUniqueViolation_PgError_23505(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	assert.True(t, isUniqueViolation(pgErr))
}

func TestIsUniqueViolation_PgError_OtherCode(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503"}
	assert.False(t, isUniqueViolation(pgErr))
}

func TestIsUniqueViolation_NonPgError(t *testing.T) {
	assert.False(t, isUniqueViolation(errors.New("some random error")))
}

func TestIsUniqueViolation_NilError(t *testing.T) {
	assert.False(t, isUniqueViolation(nil))
}
