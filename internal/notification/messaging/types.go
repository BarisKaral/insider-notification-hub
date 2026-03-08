package messaging

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NotificationConsumerConfig holds configuration for the consumer.
type NotificationConsumerConfig struct {
	Concurrency int
	RateLimit   int
	MaxRetries  int
	RetryTTL    time.Duration
}

// consumerPayload mirrors the message format published by the producer.
type consumerPayload struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Channel   string `json:"channel"`
	Content   string `json:"content"`
	Priority  string `json:"priority"`
}

// messagePayload is the message format published to the notification exchange.
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
