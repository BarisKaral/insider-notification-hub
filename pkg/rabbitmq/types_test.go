package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrRabbitMQConnectionFailed(t *testing.T) {
	assert.NotNil(t, ErrRabbitMQConnectionFailed)
	assert.Equal(t, "rabbitmq connection failed", ErrRabbitMQConnectionFailed.Error())
}

func TestErrRabbitMQChannelFailed(t *testing.T) {
	assert.NotNil(t, ErrRabbitMQChannelFailed)
	assert.Equal(t, "rabbitmq channel creation failed", ErrRabbitMQChannelFailed.Error())
}

func TestRabbitMQErrors_AreDistinct(t *testing.T) {
	assert.NotEqual(t, ErrRabbitMQConnectionFailed, ErrRabbitMQChannelFailed)
}

func TestRabbitMQConfig_StructCreation(t *testing.T) {
	config := RabbitMQConfig{
		URL: "amqp://guest:guest@localhost:5672/",
	}

	assert.Equal(t, "amqp://guest:guest@localhost:5672/", config.URL)
}

func TestRabbitMQConfig_EmptyURL(t *testing.T) {
	config := RabbitMQConfig{}
	assert.Empty(t, config.URL)
}

func TestRabbitMQQueueConfig_StructCreation(t *testing.T) {
	config := RabbitMQQueueConfig{
		Name:        "notifications",
		Exchange:    "notification-exchange",
		RoutingKey:  "notification.send",
		DLX:         "notification-dlx",
		TTL:         30000,
		MaxPriority: 10,
	}

	assert.Equal(t, "notifications", config.Name)
	assert.Equal(t, "notification-exchange", config.Exchange)
	assert.Equal(t, "notification.send", config.RoutingKey)
	assert.Equal(t, "notification-dlx", config.DLX)
	assert.Equal(t, 30000, config.TTL)
	assert.Equal(t, 10, config.MaxPriority)
}

func TestRabbitMQQueueConfig_DefaultValues(t *testing.T) {
	config := RabbitMQQueueConfig{
		Name:       "simple-queue",
		Exchange:   "simple-exchange",
		RoutingKey: "simple.key",
	}

	assert.Equal(t, "simple-queue", config.Name)
	assert.Equal(t, "simple-exchange", config.Exchange)
	assert.Equal(t, "simple.key", config.RoutingKey)
	assert.Empty(t, config.DLX)
	assert.Zero(t, config.TTL)
	assert.Zero(t, config.MaxPriority)
}

func TestRabbitMQQueueConfig_WithTTLOnly(t *testing.T) {
	config := RabbitMQQueueConfig{
		Name:       "retry-queue",
		Exchange:   "retry-exchange",
		RoutingKey: "retry.key",
		TTL:        60000,
	}

	assert.Equal(t, 60000, config.TTL)
	assert.Zero(t, config.MaxPriority)
}

func TestRabbitMQQueueConfig_ZeroTTLMeansNoTTL(t *testing.T) {
	config := RabbitMQQueueConfig{
		Name: "no-ttl-queue",
		TTL:  0,
	}

	assert.Zero(t, config.TTL)
}

func TestRabbitMQQueueConfig_ZeroMaxPriorityMeansNoPriority(t *testing.T) {
	config := RabbitMQQueueConfig{
		Name:        "no-priority-queue",
		MaxPriority: 0,
	}

	assert.Zero(t, config.MaxPriority)
}
