package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/time/rate"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/metrics"
	"github.com/baris/notification-hub/internal/provider"
	"github.com/baris/notification-hub/pkg/logger"
)

// NotificationConsumer processes notifications from RabbitMQ queues.
type NotificationConsumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// NotificationConsumerConfig holds configuration for the consumer.
type NotificationConsumerConfig struct {
	Concurrency int
	RateLimit   int
	MaxRetries  int
	RetryTTL    time.Duration
}

type notificationConsumer struct {
	service  domain.NotificationService
	provider provider.ProviderClient
	channel  *amqp.Channel
	wsHub    domain.StatusBroadcaster
	metrics  *metrics.NotificationMetrics
	config   NotificationConsumerConfig
}

var _ NotificationConsumer = (*notificationConsumer)(nil)

// consumerPayload mirrors the message format published by the producer.
type consumerPayload struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Channel   string `json:"channel"`
	Content   string `json:"content"`
	Priority  string `json:"priority"`
}

// NewNotificationConsumer creates a new consumer that processes notifications from RabbitMQ.
func NewNotificationConsumer(
	service domain.NotificationService,
	prov provider.ProviderClient,
	ch *amqp.Channel,
	wsHub domain.StatusBroadcaster,
	m *metrics.NotificationMetrics,
	cfg NotificationConsumerConfig,
) NotificationConsumer {
	return &notificationConsumer{
		service:  service,
		provider: prov,
		channel:  ch,
		wsHub:    wsHub,
		metrics:  m,
		config:   cfg,
	}
}

// Start begins consuming from main and DLQ queues for all channels.
func (c *notificationConsumer) Start(ctx context.Context) error {
	if err := c.channel.Qos(c.config.Concurrency, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	channels := []string{"sms", "email", "push"}
	for _, ch := range channels {
		if err := c.startMainConsumer(ctx, ch); err != nil {
			return err
		}
		if err := c.startDLQConsumer(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

// Stop closes the underlying AMQP channel.
func (c *notificationConsumer) Stop() error {
	return c.channel.Close()
}

func (c *notificationConsumer) startMainConsumer(ctx context.Context, channel string) error {
	queueName := fmt.Sprintf("notification.queue.%s", channel)
	consumerTag := fmt.Sprintf("main-%s", channel)

	deliveries, err := c.channel.Consume(
		queueName,   // queue
		consumerTag, // consumer tag
		false,       // autoAck
		false,       // exclusive
		false,       // noLocal
		false,       // noWait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming from %s: %w", queueName, err)
	}

	limiter := rate.NewLimiter(rate.Limit(c.config.RateLimit), c.config.RateLimit)

	for i := 0; i < c.config.Concurrency; i++ {
		go c.processMainMessages(ctx, deliveries, limiter, channel)
	}

	logger.Info().
		Str("queue", queueName).
		Int("concurrency", c.config.Concurrency).
		Int("rateLimit", c.config.RateLimit).
		Msg("main consumer started")

	return nil
}

func (c *notificationConsumer) processMainMessages(ctx context.Context, deliveries <-chan amqp.Delivery, limiter *rate.Limiter, channel string) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-deliveries:
			if !ok {
				return
			}
			c.handleMainMessage(ctx, msg, limiter, channel)
		}
	}
}

func (c *notificationConsumer) handleMainMessage(ctx context.Context, msg amqp.Delivery, limiter *rate.Limiter, channel string) {
	if err := limiter.Wait(ctx); err != nil {
		logger.Error().Err(err).Str("channel", channel).Msg("rate limiter wait failed")
		return
	}

	// Extract trace context propagated from the producer via AMQP headers.
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, amqpHeaderCarrier(msg.Headers))

	ctx, span := otel.Tracer("notification").Start(ctx, "consumer.Process")
	defer span.End()

	start := time.Now()

	var payload consumerPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		logger.Error().Err(err).Str("channel", channel).Msg("failed to unmarshal message")
		span.RecordError(err)
		span.SetStatus(codes.Error, "unmarshal failed")
		_ = msg.Nack(false, false)
		return
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		logger.Error().Err(err).Str("payloadID", payload.ID).Msg("failed to parse notification ID")
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid notification ID")
		_ = msg.Nack(false, false)
		return
	}

	span.SetAttributes(
		attribute.String("notification.id", id.String()),
		attribute.String("notification.channel", channel),
	)

	n, err := c.service.MarkAsProcessing(ctx, id)
	if err != nil {
		logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to mark as processing")
		span.RecordError(err)
		span.SetStatus(codes.Error, "mark processing failed")
		_ = msg.Nack(false, false)
		return
	}

	// Skip cancelled or already sent notifications.
	if n.Status == domain.NotificationStatusCancelled || n.Status == domain.NotificationStatusSent {
		logger.Info().
			Str("notificationID", id.String()).
			Str("status", string(n.Status)).
			Msg("skipping notification, already terminal")
		_ = msg.Ack(false)
		return
	}

	providerReq := &provider.ProviderRequest{
		To:      payload.Recipient,
		Channel: payload.Channel,
		Content: payload.Content,
	}

	resp, err := c.provider.Send(ctx, providerReq)
	if err != nil {
		logger.Error().Err(err).Str("notificationID", id.String()).Msg("provider send failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, "provider send failed")
		// NACK without requeue — message goes to DLQ via DLX.
		// DLQ consumer handles retry/failure status transitions.
		_ = msg.Nack(false, false)
		return
	}

	if err := c.service.MarkAsSent(ctx, id, resp.MessageID); err != nil {
		logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to mark as sent")
	}

	if c.metrics != nil {
		c.metrics.IncTotal(channel, string(domain.NotificationStatusSent))
		c.metrics.ObserveDuration(channel, time.Since(start))
	}

	c.broadcastStatus(n, string(domain.NotificationStatusSent))

	_ = msg.Ack(false)

	logger.Info().
		Str("notificationID", id.String()).
		Str("providerMsgID", resp.MessageID).
		Msg("notification sent successfully")
}

func (c *notificationConsumer) startDLQConsumer(ctx context.Context, channel string) error {
	queueName := fmt.Sprintf("notification.dlq.%s", channel)
	consumerTag := fmt.Sprintf("dlq-%s", channel)

	deliveries, err := c.channel.Consume(
		queueName,   // queue
		consumerTag, // consumer tag
		false,       // autoAck
		false,       // exclusive
		false,       // noLocal
		false,       // noWait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming from %s: %w", queueName, err)
	}

	go c.processDLQMessages(ctx, deliveries, channel)

	logger.Info().
		Str("queue", queueName).
		Msg("DLQ consumer started")

	return nil
}

func (c *notificationConsumer) processDLQMessages(ctx context.Context, deliveries <-chan amqp.Delivery, channel string) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-deliveries:
			if !ok {
				return
			}
			c.handleDLQMessage(ctx, msg, channel)
		}
	}
}

func (c *notificationConsumer) handleDLQMessage(ctx context.Context, msg amqp.Delivery, channel string) {
	var payload consumerPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		logger.Error().Err(err).Str("channel", channel).Msg("failed to unmarshal DLQ message")
		_ = msg.Nack(false, false)
		return
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		logger.Error().Err(err).Str("payloadID", payload.ID).Msg("failed to parse notification ID from DLQ")
		_ = msg.Nack(false, false)
		return
	}

	var retryCount int32
	if msg.Headers != nil {
		if rc, ok := msg.Headers["x-retry-count"]; ok {
			if v, ok := rc.(int32); ok {
				retryCount = v
			}
		}
	}

	if int(retryCount) < c.config.MaxRetries {
		if err := c.service.MarkAsRetrying(ctx, id); err != nil {
			logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to mark as retrying")
			_ = msg.Nack(false, false)
			return
		}

		newRetryCount := retryCount + 1

		headers := amqp.Table{
			"x-retry-count": newRetryCount,
		}

		// Forward trace context headers from the original message.
		if msg.Headers != nil {
			for _, key := range []string{"traceparent", "tracestate"} {
				if v, ok := msg.Headers[key]; ok {
					headers[key] = v
				}
			}
		}

		err := c.channel.PublishWithContext(ctx,
			"notification.retry.exchange", // exchange
			channel,                       // routing key
			false,                         // mandatory
			false,                         // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Priority:     msg.Priority,
				MessageId:    msg.MessageId,
				Body:         msg.Body,
				Headers:      headers,
			},
		)
		if err != nil {
			logger.Error().Err(err).
				Str("notificationID", id.String()).
				Int32("retryCount", newRetryCount).
				Msg("failed to publish to retry exchange")
			_ = msg.Nack(false, false)
			return
		}

		_ = msg.Ack(false)

		logger.Info().
			Str("notificationID", id.String()).
			Int32("retryCount", newRetryCount).
			Msg("notification sent to retry queue")
	} else {
		if err := c.service.MarkAsFailed(ctx, id, "max retries exceeded", int(retryCount)); err != nil {
			logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to mark as permanently failed")
		}

		if c.metrics != nil {
			c.metrics.IncTotal(channel, string(domain.NotificationStatusFailed))
		}

		// Fetch the notification to get batch ID for broadcasting.
		n, fetchErr := c.service.GetByID(ctx, id)
		if fetchErr == nil {
			c.broadcastStatus(n, string(domain.NotificationStatusFailed))
		}

		_ = msg.Ack(false)

		logger.Info().
			Str("notificationID", id.String()).
			Int32("retryCount", retryCount).
			Msg("notification permanently failed, max retries exceeded")
	}
}

func (c *notificationConsumer) broadcastStatus(n *domain.Notification, status string) {
	if c.wsHub == nil {
		return
	}
	var batchID *string
	if n.BatchID != nil {
		s := n.BatchID.String()
		batchID = &s
	}
	c.wsHub.Broadcast(n.ID.String(), batchID, status)
}
