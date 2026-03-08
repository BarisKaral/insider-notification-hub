package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConnection interface {
	Channel() (*amqp.Channel, error)
	Close() error
}

type connection struct {
	amqpConnection *amqp.Connection
}

func NewRabbitMQConnection(config RabbitMQConfig) (RabbitMQConnection, error) {
	amqpConnection, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRabbitMQConnectionFailed, err)
	}
	return &connection{amqpConnection: amqpConnection}, nil
}

func (c *connection) Channel() (*amqp.Channel, error) {
	channel, err := c.amqpConnection.Channel()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRabbitMQChannelFailed, err)
	}
	return channel, nil
}

func (c *connection) Close() error {
	return c.amqpConnection.Close()
}
