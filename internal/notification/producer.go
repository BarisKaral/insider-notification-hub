package notification

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NotificationProducer publishes notifications to RabbitMQ channel-specific queues.
type NotificationProducer interface {
	Publish(ctx context.Context, n *Notification) error
	PublishBatch(ctx context.Context, notifications []*Notification) error
}

type notificationProducer struct {
	channel *amqp.Channel
}

var _ NotificationProducer = (*notificationProducer)(nil)

// NewNotificationProducer creates a new producer that publishes to the notification exchange.
func NewNotificationProducer(ch *amqp.Channel) NotificationProducer {
	return &notificationProducer{channel: ch}
}

type messagePayload struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Channel   string `json:"channel"`
	Content   string `json:"content"`
	Priority  string `json:"priority"`
}

// Publish marshals a notification to JSON and publishes it to the notification exchange
// with the channel as routing key.
func (p *notificationProducer) Publish(ctx context.Context, n *Notification) error {
	payload := messagePayload{
		ID:        n.ID.String(),
		Recipient: n.Recipient,
		Channel:   string(n.Channel),
		Content:   n.Content,
		Priority:  string(n.Priority),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification %s: %w", n.ID, err)
	}

	err = p.channel.PublishWithContext(ctx,
		"notification.exchange", // exchange
		string(n.Channel),      // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Priority:     n.Priority.ToUint8(),
			MessageId:    n.ID.String(),
			Body:         body,
			Headers: amqp.Table{
				"x-retry-count": int32(0),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish notification %s: %w", n.ID, err)
	}

	return nil
}

// PublishBatch publishes multiple notifications. It stops and returns the error
// if any individual publish fails.
func (p *notificationProducer) PublishBatch(ctx context.Context, notifications []*Notification) error {
	for _, n := range notifications {
		if err := p.Publish(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
