package messaging

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmqpHeaderCarrier_Get_ExistingStringKey(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{
		"traceparent": "00-abc123-def456-01",
	})

	result := carrier.Get("traceparent")

	assert.Equal(t, "00-abc123-def456-01", result)
}

func TestAmqpHeaderCarrier_Get_NonExistentKey(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{
		"traceparent": "00-abc123-def456-01",
	})

	result := carrier.Get("missing-key")

	assert.Equal(t, "", result)
}

func TestAmqpHeaderCarrier_Get_NonStringValue(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{
		"int-value": int32(42),
	})

	result := carrier.Get("int-value")

	assert.Equal(t, "", result)
}

func TestAmqpHeaderCarrier_Get_EmptyTable(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{})

	result := carrier.Get("anything")

	assert.Equal(t, "", result)
}

func TestAmqpHeaderCarrier_Set(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{})

	carrier.Set("traceparent", "00-abc123-def456-01")

	assert.Equal(t, "00-abc123-def456-01", (amqp.Table)(carrier)["traceparent"])
}

func TestAmqpHeaderCarrier_Set_OverwritesExisting(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{
		"key": "old-value",
	})

	carrier.Set("key", "new-value")

	assert.Equal(t, "new-value", (amqp.Table)(carrier)["key"])
}

func TestAmqpHeaderCarrier_Keys_ReturnsAllKeys(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{
		"traceparent": "value1",
		"tracestate":  "value2",
		"custom":      "value3",
	})

	keys := carrier.Keys()

	sort.Strings(keys)
	assert.Equal(t, []string{"custom", "traceparent", "tracestate"}, keys)
}

func TestAmqpHeaderCarrier_Keys_EmptyTable(t *testing.T) {
	carrier := amqpHeaderCarrier(amqp.Table{})

	keys := carrier.Keys()

	assert.NotNil(t, keys)
	assert.Empty(t, keys)
}

func TestConsumerPayload_JSONMarshal(t *testing.T) {
	payload := consumerPayload{
		ID:        "notif-123",
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   "Hello World",
		Priority:  "high",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "notif-123", result["id"])
	assert.Equal(t, "user@example.com", result["recipient"])
	assert.Equal(t, "email", result["channel"])
	assert.Equal(t, "Hello World", result["content"])
	assert.Equal(t, "high", result["priority"])
}

func TestConsumerPayload_JSONUnmarshal(t *testing.T) {
	jsonStr := `{"id":"notif-456","recipient":"+1234567890","channel":"sms","content":"Test message","priority":"low"}`

	var payload consumerPayload
	err := json.Unmarshal([]byte(jsonStr), &payload)
	require.NoError(t, err)

	assert.Equal(t, "notif-456", payload.ID)
	assert.Equal(t, "+1234567890", payload.Recipient)
	assert.Equal(t, "sms", payload.Channel)
	assert.Equal(t, "Test message", payload.Content)
	assert.Equal(t, "low", payload.Priority)
}

func TestMessagePayload_JSONMarshal(t *testing.T) {
	payload := messagePayload{
		ID:        "msg-789",
		Recipient: "device-token-abc",
		Channel:   "push",
		Content:   "Push notification content",
		Priority:  "normal",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "msg-789", result["id"])
	assert.Equal(t, "device-token-abc", result["recipient"])
	assert.Equal(t, "push", result["channel"])
	assert.Equal(t, "Push notification content", result["content"])
	assert.Equal(t, "normal", result["priority"])
}

func TestMessagePayload_JSONUnmarshal(t *testing.T) {
	jsonStr := `{"id":"msg-101","recipient":"user@test.com","channel":"email","content":"Email body","priority":"high"}`

	var payload messagePayload
	err := json.Unmarshal([]byte(jsonStr), &payload)
	require.NoError(t, err)

	assert.Equal(t, "msg-101", payload.ID)
	assert.Equal(t, "user@test.com", payload.Recipient)
	assert.Equal(t, "email", payload.Channel)
	assert.Equal(t, "Email body", payload.Content)
	assert.Equal(t, "high", payload.Priority)
}

func TestMessagePayload_JSONRoundTrip(t *testing.T) {
	original := messagePayload{
		ID:        "round-trip-1",
		Recipient: "recipient@example.com",
		Channel:   "sms",
		Content:   "Round trip content",
		Priority:  "low",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded messagePayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestNotificationConsumerConfig_FieldAccess(t *testing.T) {
	config := NotificationConsumerConfig{
		Concurrency: 5,
		RateLimit:   100,
		MaxRetries:  3,
		RetryTTL:    30 * time.Second,
	}

	assert.Equal(t, 5, config.Concurrency)
	assert.Equal(t, 100, config.RateLimit)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 30*time.Second, config.RetryTTL)
}

func TestNotificationConsumerConfig_ZeroValue(t *testing.T) {
	var config NotificationConsumerConfig

	assert.Equal(t, 0, config.Concurrency)
	assert.Equal(t, 0, config.RateLimit)
	assert.Equal(t, 0, config.MaxRetries)
	assert.Equal(t, time.Duration(0), config.RetryTTL)
}
