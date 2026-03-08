package health

import (
	"context"

	"github.com/bariskaral/insider-notification-hub/pkg/rabbitmq"
	"gorm.io/gorm"
)

// HealthService defines the interface for health checking.
type HealthService interface {
	Check(ctx context.Context) HealthResponse
}

type healthService struct {
	database            *gorm.DB
	rabbitMQConnection rabbitmq.RabbitMQConnection
}

var _ HealthService = (*healthService)(nil)

// NewHealthService creates a new HealthService.
func NewHealthService(database *gorm.DB, rabbitMQConnection rabbitmq.RabbitMQConnection) HealthService {
	return &healthService{
		database:            database,
		rabbitMQConnection: rabbitMQConnection,
	}
}

// Check performs health checks on DB and RabbitMQ dependencies.
func (s *healthService) Check(ctx context.Context) HealthResponse {
	response := HealthResponse{
		Status: "healthy",
		Checks: make(map[string]string),
	}

	// Check database connectivity.
	var result int
	if err := s.database.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		response.Status = "unhealthy"
		response.Checks["database"] = "down"
	} else {
		response.Checks["database"] = "up"
	}

	// Check RabbitMQ connectivity by opening and closing a channel.
	ch, err := s.rabbitMQConnection.Channel()
	if err != nil {
		response.Status = "unhealthy"
		response.Checks["rabbitmq"] = "down"
	} else {
		if ch != nil {
			ch.Close()
		}
		response.Checks["rabbitmq"] = "up"
	}

	return response
}
