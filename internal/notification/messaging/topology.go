package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupNotificationQueues declares all notification-specific exchanges, queues, and bindings.
func SetupNotificationQueues(ch *amqp.Channel) error {
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
			return fmt.Errorf("failed to declare exchange %s: %w", ex.name, err)
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
			return fmt.Errorf("failed to declare queue %s: %w", mainQueue, err)
		}
		if err := ch.QueueBind(mainQueue, channel, "notification.exchange", false, nil); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", mainQueue, err)
		}

		// DLQ
		dlq := fmt.Sprintf("notification.dlq.%s", channel)
		if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", dlq, err)
		}
		if err := ch.QueueBind(dlq, channel, "notification.dlx", false, nil); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", dlq, err)
		}

		// Retry queue with TTL, DLX back to main exchange
		retryQueue := fmt.Sprintf("notification.retry.%s", channel)
		if _, err := ch.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange":    "notification.exchange",
			"x-dead-letter-routing-key": channel,
			"x-message-ttl":             int32(30000),
		}); err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", retryQueue, err)
		}
		if err := ch.QueueBind(retryQueue, channel, "notification.retry.exchange", false, nil); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", retryQueue, err)
		}
	}

	return nil
}
