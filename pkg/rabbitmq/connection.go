package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection interface {
	Channel() (*amqp.Channel, error)
	Close() error
}

type Config struct {
	URL string
}

type connection struct {
	conn *amqp.Connection
}

func NewConnection(cfg Config) (Connection, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	return &connection{conn: conn}, nil
}

func (c *connection) Channel() (*amqp.Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrChannelFailed, err)
	}
	return ch, nil
}

func (c *connection) Close() error {
	return c.conn.Close()
}

func SetupQueues(ch *amqp.Channel) error {
	// Declare exchanges
	exchanges := []struct {
		name string
		kind string
	}{
		{"notification.exchange", "direct"},
		{"notification.dlx", "direct"},
		{"notification.retry.exchange", "direct"},
	}

	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex.name, ex.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("%w: exchange %s: %v", ErrQueueSetupFailed, ex.name, err)
		}
	}

	channels := []string{"sms", "email", "push"}

	for _, channel := range channels {
		// Main queue with DLX and priority
		mainQueue := fmt.Sprintf("notification.queue.%s", channel)
		_, err := ch.QueueDeclare(mainQueue, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange":    "notification.dlx",
			"x-dead-letter-routing-key": channel,
			"x-max-priority":            int32(3),
		})
		if err != nil {
			return fmt.Errorf("%w: queue %s: %v", ErrQueueSetupFailed, mainQueue, err)
		}
		if err := ch.QueueBind(mainQueue, channel, "notification.exchange", false, nil); err != nil {
			return fmt.Errorf("%w: bind %s: %v", ErrQueueSetupFailed, mainQueue, err)
		}

		// DLQ
		dlq := fmt.Sprintf("notification.dlq.%s", channel)
		_, err = ch.QueueDeclare(dlq, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("%w: queue %s: %v", ErrQueueSetupFailed, dlq, err)
		}
		if err := ch.QueueBind(dlq, channel, "notification.dlx", false, nil); err != nil {
			return fmt.Errorf("%w: bind %s: %v", ErrQueueSetupFailed, dlq, err)
		}

		// Retry queue with TTL, DLX back to main exchange
		retryQueue := fmt.Sprintf("notification.retry.%s", channel)
		_, err = ch.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange":    "notification.exchange",
			"x-dead-letter-routing-key": channel,
			"x-message-ttl":             int32(30000),
		})
		if err != nil {
			return fmt.Errorf("%w: queue %s: %v", ErrQueueSetupFailed, retryQueue, err)
		}
		if err := ch.QueueBind(retryQueue, channel, "notification.retry.exchange", false, nil); err != nil {
			return fmt.Errorf("%w: bind %s: %v", ErrQueueSetupFailed, retryQueue, err)
		}
	}

	return nil
}
