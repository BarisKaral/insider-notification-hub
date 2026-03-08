package notification

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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

// amqpHeaderCarrier adapts amqp.Table for OpenTelemetry trace context propagation.
type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	if v, ok := (amqp.Table)(c)[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpHeaderCarrier) Set(key, value string) {
	(amqp.Table)(c)[key] = value
}

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len((amqp.Table)(c)))
	for k := range (amqp.Table)(c) {
		keys = append(keys, k)
	}
	return keys
}

// Publish marshals a notification to JSON and publishes it to the notification exchange
// with the channel as routing key.
func (p *notificationProducer) Publish(ctx context.Context, n *Notification) error {
	ctx, span := otel.Tracer("notification").Start(ctx, "producer.Publish")
	defer span.End()

	payload := messagePayload{
		ID:        n.ID.String(),
		Recipient: n.Recipient,
		Channel:   string(n.Channel),
		Content:   n.Content,
		Priority:  string(n.Priority),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal failed")
		return fmt.Errorf("failed to marshal notification %s: %w", n.ID, err)
	}

	headers := amqp.Table{
		"x-retry-count": int32(0),
	}

	// Inject trace context into AMQP headers for propagation to consumer.
	otel.GetTextMapPropagator().Inject(ctx, amqpHeaderCarrier(headers))

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
			Headers:      headers,
		},
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
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
