package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConnection interface {
	Channel() (*amqp.Channel, error)
	Close() error
}

type RabbitMQConfig struct {
	URL string
}

type connection struct {
	conn *amqp.Connection
}

func NewRabbitMQConnection(cfg RabbitMQConfig) (RabbitMQConnection, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRabbitMQConnectionFailed, err)
	}
	return &connection{conn: conn}, nil
}

func (c *connection) Channel() (*amqp.Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRabbitMQChannelFailed, err)
	}
	return ch, nil
}

func (c *connection) Close() error {
	return c.conn.Close()
}

