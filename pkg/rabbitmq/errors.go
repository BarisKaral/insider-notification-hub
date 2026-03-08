package rabbitmq

import "errors"

var (
	ErrRabbitMQConnectionFailed = errors.New("rabbitmq connection failed")
	ErrRabbitMQChannelFailed    = errors.New("rabbitmq channel creation failed")
)
