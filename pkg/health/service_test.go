package health

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Mocks ---

type mockRabbitMQConnection struct {
	mock.Mock
}

func (m *mockRabbitMQConnection) Channel() (*amqp.Channel, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*amqp.Channel), args.Error(1)
}

func (m *mockRabbitMQConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

// --- Helpers ---

func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mockDB, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return gormDB, mockDB
}

// --- Tests ---

func TestHealthService_Check_AllHealthy(t *testing.T) {
	db, mockDB := newTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	mockDB.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	mockRMQ := new(mockRabbitMQConnection)
	mockRMQ.On("Channel").Return(nil, nil)

	svc := NewHealthService(db, mockRMQ)
	resp := svc.Check(context.Background())

	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "up", resp.Checks["database"])
	assert.Equal(t, "up", resp.Checks["rabbitmq"])

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockRMQ.AssertExpectations(t)
}

func TestHealthService_Check_DBDown(t *testing.T) {
	db, mockDB := newTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	mockDB.ExpectQuery("SELECT 1").WillReturnError(assert.AnError)

	mockRMQ := new(mockRabbitMQConnection)
	mockRMQ.On("Channel").Return(nil, nil)

	svc := NewHealthService(db, mockRMQ)
	resp := svc.Check(context.Background())

	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "down", resp.Checks["database"])
	assert.Equal(t, "up", resp.Checks["rabbitmq"])

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockRMQ.AssertExpectations(t)
}

func TestHealthService_Check_RabbitMQDown(t *testing.T) {
	db, mockDB := newTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	mockDB.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	mockRMQ := new(mockRabbitMQConnection)
	mockRMQ.On("Channel").Return(nil, assert.AnError)

	svc := NewHealthService(db, mockRMQ)
	resp := svc.Check(context.Background())

	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "up", resp.Checks["database"])
	assert.Equal(t, "down", resp.Checks["rabbitmq"])

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockRMQ.AssertExpectations(t)
}

func TestHealthService_Check_AllDown(t *testing.T) {
	db, mockDB := newTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	mockDB.ExpectQuery("SELECT 1").WillReturnError(assert.AnError)

	mockRMQ := new(mockRabbitMQConnection)
	mockRMQ.On("Channel").Return(nil, assert.AnError)

	svc := NewHealthService(db, mockRMQ)
	resp := svc.Check(context.Background())

	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "down", resp.Checks["database"])
	assert.Equal(t, "down", resp.Checks["rabbitmq"])

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockRMQ.AssertExpectations(t)
}
