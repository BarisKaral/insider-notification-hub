package config

import "errors"

var (
	ErrMissingAppPort        = errors.New("APP_PORT is required")
	ErrMissingDatabaseConfig = errors.New("database configuration is incomplete")
	ErrMissingRabbitMQURL    = errors.New("RABBITMQ_URL is required")
)
