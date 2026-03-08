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
	"golang.org/x/time/rate"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/metrics"
	"github.com/baris/notification-hub/internal/notification/service"
	"github.com/baris/notification-hub/internal/notification/ws"
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
	processingService service.NotificationProcessingService
	channel           *amqp.Channel
	wsHub             ws.StatusBroadcaster
	metrics           *metrics.NotificationMetrics
	config            NotificationConsumerConfig
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
	processingSvc service.NotificationProcessingService,
	ch *amqp.Channel,
	wsHub ws.StatusBroadcaster,
	m *metrics.NotificationMetrics,
	cfg NotificationConsumerConfig,
) NotificationConsumer {
	return &notificationConsumer{
		processingService: processingSvc,
		channel:           ch,
		wsHub:             wsHub,
		metrics:           m,
		config:            cfg,
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

	ctx, span := otel.Tracer("notification-hub").Start(ctx, fmt.Sprintf("consumer.process.%s", channel))
	defer span.End()

	start := time.Now()

	var payload consumerPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		logger.Error().Err(err).Str("channel", channel).Msg("failed to unmarshal message")
		_ = msg.Nack(false, false)
		return
	}

	id, err := uuid.Parse(payload.ID)
	if err != nil {
		logger.Error().Err(err).Str("payloadID", payload.ID).Msg("failed to parse notification ID")
		_ = msg.Nack(false, false)
		return
	}

	span.SetAttributes(
		attribute.String("notification.id", id.String()),
		attribute.String("notification.channel", channel),
	)

	n, sent, err := c.processingService.ProcessAndSend(ctx, id)
	if err != nil {
		logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to process notification")
		_ = msg.Nack(false, false)
		return
	}

	if !sent {
		logger.Info().
			Str("notificationID", id.String()).
			Str("status", string(n.Status)).
			Msg("skipping notification, already terminal")
		_ = msg.Ack(false)
		return
	}

	if c.metrics != nil {
		c.metrics.IncTotal(channel, string(domain.NotificationStatusSent))
		c.metrics.ObserveDuration(channel, time.Since(start))
	}

	c.broadcastStatus(n, string(domain.NotificationStatusSent))

	_ = msg.Ack(false)

	logger.Info().
		Str("notificationID", id.String()).
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

	n, err := c.processingService.HandleDeliveryFailure(ctx, id, int(retryCount), c.config.MaxRetries)
	if err != nil {
		logger.Error().Err(err).Str("notificationID", id.String()).Msg("failed to handle delivery failure")
		_ = msg.Nack(false, false)
		return
	}

	if n.Status == domain.NotificationStatusFailed {
		if c.metrics != nil {
			c.metrics.IncTotal(channel, string(domain.NotificationStatusFailed))
		}
		c.broadcastStatus(n, string(domain.NotificationStatusFailed))

		logger.Info().
			Str("notificationID", id.String()).
			Int32("retryCount", retryCount).
			Msg("notification permanently failed, max retries exceeded")
	} else {
		logger.Info().
			Str("notificationID", id.String()).
			Int32("retryCount", retryCount+1).
			Msg("notification sent to retry queue")
	}

	_ = msg.Ack(false)
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
