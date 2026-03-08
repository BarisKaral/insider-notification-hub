package health

import (
	"context"

	"github.com/baris/notification-hub/pkg/rabbitmq"
	"gorm.io/gorm"
)

// HealthService defines the interface for health checking.
type HealthService interface {
	Check(ctx context.Context) HealthResponse
}

type healthService struct {
	db       *gorm.DB
	rabbitmq rabbitmq.RabbitMQConnection
}

var _ HealthService = (*healthService)(nil)

// NewHealthService creates a new HealthService.
func NewHealthService(db *gorm.DB, rabbitmq rabbitmq.RabbitMQConnection) HealthService {
	return &healthService{
		db:       db,
		rabbitmq: rabbitmq,
	}
}

// Check performs health checks on DB and RabbitMQ dependencies.
func (s *healthService) Check(ctx context.Context) HealthResponse {
	resp := HealthResponse{
		Status: "healthy",
		Checks: make(map[string]string),
	}

	// Check database connectivity.
	var result int
	if err := s.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		resp.Status = "unhealthy"
		resp.Checks["database"] = "down"
	} else {
		resp.Checks["database"] = "up"
	}

	// Check RabbitMQ connectivity by opening and closing a channel.
	ch, err := s.rabbitmq.Channel()
	if err != nil {
		resp.Status = "unhealthy"
		resp.Checks["rabbitmq"] = "down"
	} else {
		if ch != nil {
			ch.Close()
		}
		resp.Checks["rabbitmq"] = "up"
	}

	return resp
}
