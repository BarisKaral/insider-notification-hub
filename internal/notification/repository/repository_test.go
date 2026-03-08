package repository

import (
	"context"
	"database/sql/driver"
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

	"github.com/bariskaral/insider-notification-hub/internal/notification/domain"
)

// ===================== Helpers =====================

func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

// notificationColumns returns the column names for sqlmock.NewRows in the order GORM scans them.
var notificationColumns = []string{
	"id", "recipient", "channel", "content", "priority", "status",
	"batch_id", "idempotency_key", "template_id", "template_vars",
	"provider_msg_id", "retry_count", "scheduled_at", "sent_at",
	"failed_at", "failure_reason", "created_at", "updated_at", "deleted_at",
}

func newTestNotification() *domain.Notification {
	return &domain.Notification{
		ID:        uuid.New(),
		Recipient: "+905551234567",
		Channel:   domain.NotificationChannelSMS,
		Content:   "Hello World",
		Priority:  domain.NotificationPriorityNormal,
		Status:    domain.NotificationStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func notificationRow(n *domain.Notification) *sqlmock.Rows {
	return sqlmock.NewRows(notificationColumns).
		AddRow(
			n.ID, n.Recipient, n.Channel, n.Content, n.Priority, n.Status,
			n.BatchID, n.IdempotencyKey, n.TemplateID, n.TemplateVars,
			n.ProviderMsgID, n.RetryCount, n.ScheduledAt, n.SentAt,
			n.FailedAt, n.FailureReason, n.CreatedAt, n.UpdatedAt, nil,
		)
}

// anyArgs returns a slice of N sqlmock.AnyArg values for use with WithArgs(...).
func anyArgs(n int) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	return args
}

// ===================== Existing Tests =====================

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

// ===================== Create Tests =====================

func TestCreate_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()
	n := newTestNotification()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notifications"`).
		WithArgs(anyArgs(18)...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(n.ID))
	mock.ExpectCommit()

	err := repo.Create(ctx, n)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_UniqueViolation(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()
	n := newTestNotification()
	idempKey := "dup-key-123"
	n.IdempotencyKey = &idempKey

	pgErr := &pgconn.PgError{Code: "23505"}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notifications"`).
		WithArgs(anyArgs(18)...).
		WillReturnError(pgErr)
	mock.ExpectRollback()

	err := repo.Create(ctx, n)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DUPLICATE_IDEMPOTENCY_KEY")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_GenericError(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()
	n := newTestNotification()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notifications"`).
		WithArgs(anyArgs(18)...).
		WillReturnError(errors.New("connection refused"))
	mock.ExpectRollback()

	err := repo.Create(ctx, n)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NOTIFICATION_CREATE_FAILED")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== CreateBatch Tests =====================

func TestCreateBatch_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n1 := newTestNotification()
	n2 := newTestNotification()
	n2.Recipient = "user@example.com"
	n2.Channel = domain.NotificationChannelEmail

	mock.ExpectBegin()
	// GORM batch insert generates a single INSERT with multiple value groups
	// 18 args per row (template_vars is inlined as NULL) * 2 rows = 36
	mock.ExpectQuery(`INSERT INTO "notifications"`).
		WithArgs(anyArgs(36)...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(n1.ID).AddRow(n2.ID))
	mock.ExpectCommit()

	err := repo.CreateBatch(ctx, []*domain.Notification{n1, n2})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateBatch_UniqueViolation(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n1 := newTestNotification()
	n2 := newTestNotification()

	pgErr := &pgconn.PgError{Code: "23505"}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notifications"`).
		WithArgs(anyArgs(36)...).
		WillReturnError(pgErr)
	mock.ExpectRollback()

	err := repo.CreateBatch(ctx, []*domain.Notification{n1, n2})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DUPLICATE_IDEMPOTENCY_KEY")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetByID Tests =====================

func TestGetByID_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n := newTestNotification()
	rows := notificationRow(n)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE id = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(n.ID, 1).
		WillReturnRows(rows)

	result, err := repo.GetByID(ctx, n.ID)

	require.NoError(t, err)
	assert.Equal(t, n.ID, result.ID)
	assert.Equal(t, n.Recipient, result.Recipient)
	assert.Equal(t, n.Channel, result.Channel)
	assert.Equal(t, n.Content, result.Content)
	assert.Equal(t, n.Status, result.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE id = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	result, err := repo.GetByID(ctx, id)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Equal(t, domain.ErrNotificationNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_DBError(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE id = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(id, 1).
		WillReturnError(errors.New("connection timeout"))

	result, err := repo.GetByID(ctx, id)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NOTIFICATION_NOT_FOUND")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetByBatchID Tests =====================

func TestGetByBatchID_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	batchID := uuid.New()
	n1 := newTestNotification()
	n1.BatchID = &batchID
	n2 := newTestNotification()
	n2.BatchID = &batchID
	n2.Recipient = "user@example.com"
	n2.Channel = domain.NotificationChannelEmail

	rows := sqlmock.NewRows(notificationColumns).
		AddRow(
			n1.ID, n1.Recipient, n1.Channel, n1.Content, n1.Priority, n1.Status,
			n1.BatchID, n1.IdempotencyKey, n1.TemplateID, n1.TemplateVars,
			n1.ProviderMsgID, n1.RetryCount, n1.ScheduledAt, n1.SentAt,
			n1.FailedAt, n1.FailureReason, n1.CreatedAt, n1.UpdatedAt, nil,
		).
		AddRow(
			n2.ID, n2.Recipient, n2.Channel, n2.Content, n2.Priority, n2.Status,
			n2.BatchID, n2.IdempotencyKey, n2.TemplateID, n2.TemplateVars,
			n2.ProviderMsgID, n2.RetryCount, n2.ScheduledAt, n2.SentAt,
			n2.FailedAt, n2.FailureReason, n2.CreatedAt, n2.UpdatedAt, nil,
		)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE batch_id = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(batchID).
		WillReturnRows(rows)

	results, err := repo.GetByBatchID(ctx, batchID)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, n1.ID, results[0].ID)
	assert.Equal(t, n2.ID, results[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByBatchID_Error(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	batchID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE batch_id = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(batchID).
		WillReturnError(errors.New("db error"))

	results, err := repo.GetByBatchID(ctx, batchID)

	assert.Nil(t, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NOTIFICATION_NOT_FOUND")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== List Tests =====================

func TestList_Success_NoFilters(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n1 := newTestNotification()
	n2 := newTestNotification()
	n2.Recipient = "user@example.com"
	n2.Channel = domain.NotificationChannelEmail

	// Count query
	mock.ExpectQuery(`SELECT count\(\*\) FROM "notifications" WHERE "notifications"\."deleted_at" IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Find query
	rows := sqlmock.NewRows(notificationColumns).
		AddRow(
			n1.ID, n1.Recipient, n1.Channel, n1.Content, n1.Priority, n1.Status,
			n1.BatchID, n1.IdempotencyKey, n1.TemplateID, n1.TemplateVars,
			n1.ProviderMsgID, n1.RetryCount, n1.ScheduledAt, n1.SentAt,
			n1.FailedAt, n1.FailureReason, n1.CreatedAt, n1.UpdatedAt, nil,
		).
		AddRow(
			n2.ID, n2.Recipient, n2.Channel, n2.Content, n2.Priority, n2.Status,
			n2.BatchID, n2.IdempotencyKey, n2.TemplateID, n2.TemplateVars,
			n2.ProviderMsgID, n2.RetryCount, n2.ScheduledAt, n2.SentAt,
			n2.FailedAt, n2.FailureReason, n2.CreatedAt, n2.UpdatedAt, nil,
		)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE "notifications"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`).
		WillReturnRows(rows)

	filter := domain.NotificationListFilter{Limit: 20, Offset: 0}
	results, total, err := repo.List(ctx, filter)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_Success_WithStatusFilter(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n := newTestNotification()
	n.Status = domain.NotificationStatusSent

	// Count query with status filter
	mock.ExpectQuery(`SELECT count\(\*\) FROM "notifications" WHERE status = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(string(domain.NotificationStatusSent)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Find query with status filter; GORM parameterizes LIMIT and OFFSET too
	rows := notificationRow(n)
	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE status = .+ AND "notifications"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`).
		WithArgs(string(domain.NotificationStatusSent), 20).
		WillReturnRows(rows)

	filter := domain.NotificationListFilter{
		Status: string(domain.NotificationStatusSent),
		Limit:  20,
		Offset: 0,
	}
	results, total, err := repo.List(ctx, filter)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, domain.NotificationStatusSent, results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestList_CountError(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "notifications" WHERE "notifications"\."deleted_at" IS NULL`).
		WillReturnError(errors.New("db error"))

	filter := domain.NotificationListFilter{Limit: 20, Offset: 0}
	results, total, err := repo.List(ctx, filter)

	require.Error(t, err)
	assert.Nil(t, results)
	assert.Equal(t, int64(0), total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== Update Tests =====================

func TestUpdate_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n := newTestNotification()
	n.Status = domain.NotificationStatusSent

	// GORM Save generates UPDATE with all fields.
	// template_vars (jsonb nil) is inlined as (NULL), so 18 args total:
	// 17 SET columns (excluding template_vars and id) + 1 WHERE id
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notifications" SET`).
		WithArgs(anyArgs(18)...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Update(ctx, n)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetByIdempotencyKey Tests =====================

func TestGetByIdempotencyKey_Found(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	key := "idem-key-abc"
	n := newTestNotification()
	n.IdempotencyKey = &key

	rows := notificationRow(n)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE idempotency_key = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(key, 1).
		WillReturnRows(rows)

	result, err := repo.GetByIdempotencyKey(ctx, key)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, n.ID, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByIdempotencyKey_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	key := "nonexistent-key"

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE idempotency_key = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(key, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	result, err := repo.GetByIdempotencyKey(ctx, key)

	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByIdempotencyKey_Error(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	key := "some-key"

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE idempotency_key = .+ AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(key, 1).
		WillReturnError(errors.New("connection lost"))

	result, err := repo.GetByIdempotencyKey(ctx, key)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetForProcessing Tests =====================

func TestGetForProcessing_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n := newTestNotification()
	rows := notificationRow(n)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE id = .+ AND "notifications"\."deleted_at" IS NULL .* FOR UPDATE`).
		WithArgs(n.ID, 1).
		WillReturnRows(rows)

	result, err := repo.GetForProcessing(ctx, n.ID)

	require.NoError(t, err)
	assert.Equal(t, n.ID, result.ID)
	assert.Equal(t, n.Recipient, result.Recipient)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetForProcessing_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE id = .+ AND "notifications"\."deleted_at" IS NULL .* FOR UPDATE`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	result, err := repo.GetForProcessing(ctx, id)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Equal(t, domain.ErrNotificationNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetRecoverableNotifications Tests =====================

func TestGetRecoverableNotifications_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	n := newTestNotification()
	n.Status = domain.NotificationStatusPending
	n.CreatedAt = time.Now().UTC().Add(-10 * time.Minute)

	rows := notificationRow(n)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE \(status = .+ AND created_at < .+\) AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(string(domain.NotificationStatusPending), sqlmock.AnyArg()).
		WillReturnRows(rows)

	results, err := repo.GetRecoverableNotifications(ctx, 5*time.Minute)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, n.ID, results[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ===================== GetDueScheduledNotifications Tests =====================

func TestGetDueScheduledNotifications_Success(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewNotificationRepository(gormDB)
	ctx := context.Background()

	pastTime := time.Now().UTC().Add(-5 * time.Minute)
	n := newTestNotification()
	n.Status = domain.NotificationStatusScheduled
	n.ScheduledAt = &pastTime

	rows := notificationRow(n)

	mock.ExpectQuery(`SELECT \* FROM "notifications" WHERE \(status = .+ AND scheduled_at <= .+\) AND "notifications"\."deleted_at" IS NULL`).
		WithArgs(string(domain.NotificationStatusScheduled), sqlmock.AnyArg()).
		WillReturnRows(rows)

	results, err := repo.GetDueScheduledNotifications(ctx)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, n.ID, results[0].ID)
	assert.Equal(t, domain.NotificationStatusScheduled, results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}
