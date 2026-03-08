package rabbitmq

import "errors"

var (
	ErrConnectionFailed = errors.New("rabbitmq connection failed")
	ErrChannelFailed    = errors.New("rabbitmq channel creation failed")
	ErrQueueSetupFailed = errors.New("rabbitmq queue setup failed")
)
