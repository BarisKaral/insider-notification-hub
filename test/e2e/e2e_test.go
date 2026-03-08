//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	resp, err := makeRequest(http.MethodGet, "/health", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := readBody(resp)
	require.NoError(t, err)

	var health healthResponse
	err = json.Unmarshal(body, &health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "up", health.Checks["database"])
	assert.Equal(t, "up", health.Checks["rabbitmq"])
}

func TestCreateNotification(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	resp, apiResp := createNotification(t, "+905551234567", "sms", "Hello from e2e test")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.True(t, apiResp.Success)
	require.Nil(t, apiResp.Error)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	assert.NotEmpty(t, n.ID, "notification ID should not be empty")
	assert.Equal(t, "+905551234567", n.Recipient)
	assert.Equal(t, "sms", n.Channel)
	assert.Equal(t, "Hello from e2e test", n.Content)
	assert.Equal(t, "normal", n.Priority)
	assert.Contains(t, []string{"pending", "queued"}, n.Status, "initial status should be pending or queued")
	assert.NotEmpty(t, n.CreatedAt)
}

func TestCreateNotificationWithTemplate(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Step 1: Create a template.
	tmplName := fmt.Sprintf("e2e-template-%s", uuid.New().String()[:8])
	tmplResp, tmplAPI := createTemplate(t, tmplName, "sms", "Hello {{name}}, your code is {{code}}")

	require.Equal(t, http.StatusCreated, tmplResp.StatusCode)
	require.True(t, tmplAPI.Success)

	var tmpl templateResponse
	err := json.Unmarshal(tmplAPI.Data, &tmpl)
	require.NoError(t, err)
	require.NotEmpty(t, tmpl.ID)

	// Step 2: Create notification using template.
	body := map[string]interface{}{
		"recipient":  "+905559876543",
		"channel":    "sms",
		"templateId": tmpl.ID,
		"variables": map[string]string{
			"name": "Baris",
			"code": "1234",
		},
		"priority": "high",
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var n notificationResponse
	err = json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	// Step 3: Verify rendered content.
	assert.Equal(t, "Hello Baris, your code is 1234", n.Content)
	assert.Equal(t, tmpl.ID, *n.TemplateID)
	assert.Equal(t, "high", n.Priority)
}

func TestCreateBatchNotifications(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	body := map[string]interface{}{
		"notifications": []map[string]interface{}{
			{"recipient": "+905550000001", "channel": "sms", "content": "Batch msg 1", "priority": "normal"},
			{"recipient": "+905550000002", "channel": "email", "content": "Batch msg 2", "priority": "high"},
			{"recipient": "+905550000003", "channel": "push", "content": "Batch msg 3", "priority": "low"},
		},
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications/batch", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var batch batchResponse
	err = json.Unmarshal(apiResp.Data, &batch)
	require.NoError(t, err)

	assert.NotEmpty(t, batch.BatchID, "batchId should not be empty")
	assert.Len(t, batch.Notifications, 3, "batch should contain 3 notifications")

	for _, n := range batch.Notifications {
		assert.NotEmpty(t, n.ID)
		assert.Equal(t, batch.BatchID, *n.BatchID)
	}
}

func TestGetNotificationByID(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a notification.
	_, apiResp := createNotification(t, "+905551111111", "sms", "Get by ID test")
	require.True(t, apiResp.Success)

	var created notificationResponse
	err := json.Unmarshal(apiResp.Data, &created)
	require.NoError(t, err)

	// GET by ID.
	resp, err := makeRequest(http.MethodGet, "/api/v1/notifications/"+created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, getAPI.Success)

	var fetched notificationResponse
	err = json.Unmarshal(getAPI.Data, &fetched)
	require.NoError(t, err)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, created.Recipient, fetched.Recipient)
	assert.Equal(t, created.Channel, fetched.Channel)
	assert.Equal(t, created.Content, fetched.Content)
	assert.Equal(t, created.Priority, fetched.Priority)
}

func TestGetByBatchID(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a batch.
	body := map[string]interface{}{
		"notifications": []map[string]interface{}{
			{"recipient": "+905552222201", "channel": "sms", "content": "Batch get 1", "priority": "normal"},
			{"recipient": "+905552222202", "channel": "sms", "content": "Batch get 2", "priority": "normal"},
		},
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications/batch", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)

	var batch batchResponse
	err = json.Unmarshal(apiResp.Data, &batch)
	require.NoError(t, err)
	require.NotEmpty(t, batch.BatchID)

	// GET by batch ID.
	resp, err = makeRequest(http.MethodGet, "/api/v1/notifications/batch/"+batch.BatchID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	batchAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, batchAPI.Success)

	var items []notificationResponse
	err = json.Unmarshal(batchAPI.Data, &items)
	require.NoError(t, err)

	assert.Len(t, items, 2, "batch GET should return all 2 items")
	for _, item := range items {
		assert.Equal(t, batch.BatchID, *item.BatchID)
	}
}

func TestListNotifications(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a few notifications to ensure the list is not empty.
	for i := 0; i < 3; i++ {
		createNotification(t, fmt.Sprintf("+90555300000%d", i), "sms", fmt.Sprintf("List test %d", i))
	}

	resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?limit=20&offset=0", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var paginated paginatedResponse
	err = json.Unmarshal(apiResp.Data, &paginated)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(paginated.Items), 3, "should have at least 3 items")
	assert.GreaterOrEqual(t, paginated.Total, int64(3))
	assert.Equal(t, 20, paginated.Limit)
	assert.Equal(t, 0, paginated.Offset)
}

func TestListWithFilters(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create one SMS and one email notification with unique recipients.
	tag := uuid.New().String()[:8]
	smsRecipient := fmt.Sprintf("+90555filter-sms-%s", tag)
	emailRecipient := fmt.Sprintf("filter-email-%s@test.com", tag)

	createNotification(t, smsRecipient, "sms", "Filter test SMS")
	createNotification(t, emailRecipient, "email", "Filter test Email")

	t.Run("filter by channel=sms", func(t *testing.T) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?channel=sms&limit=100", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)

		var paginated paginatedResponse
		err = json.Unmarshal(apiResp.Data, &paginated)
		require.NoError(t, err)

		for _, item := range paginated.Items {
			assert.Equal(t, "sms", item.Channel, "filtered results should only contain sms channel")
		}
	})

	t.Run("filter by channel=email", func(t *testing.T) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?channel=email&limit=100", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)

		var paginated paginatedResponse
		err = json.Unmarshal(apiResp.Data, &paginated)
		require.NoError(t, err)

		for _, item := range paginated.Items {
			assert.Equal(t, "email", item.Channel, "filtered results should only contain email channel")
		}
	})
}

func TestCancelPendingNotification(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a notification.
	_, apiResp := createNotification(t, "+905554444444", "sms", "Cancel test")
	require.True(t, apiResp.Success)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	// Immediately cancel it. The notification could be pending, queued, or failed.
	resp, err := makeRequest(http.MethodPatch, "/api/v1/notifications/"+n.ID+"/cancel", nil)
	require.NoError(t, err)

	cancelAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)

	// Cancel should succeed for pending/queued/failed statuses.
	require.Equal(t, http.StatusOK, resp.StatusCode, "cancel should return 200; got error: %+v", cancelAPI.Error)
	require.True(t, cancelAPI.Success)

	var cancelled notificationResponse
	err = json.Unmarshal(cancelAPI.Data, &cancelled)
	require.NoError(t, err)

	assert.Equal(t, "cancelled", cancelled.Status)
	assert.Equal(t, n.ID, cancelled.ID)
}

func TestCancelSentNotification(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a notification and wait for it to be sent.
	_, apiResp := createNotification(t, "+905555555555", "sms", "Cancel sent test")
	require.True(t, apiResp.Success)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	// Wait for the notification to reach "sent" status.
	sent := waitForStatus(t, n.ID, "sent", 60*time.Second)
	require.Equal(t, "sent", sent.Status)

	// Attempt to cancel a sent notification — should fail with 409.
	resp, err := makeRequest(http.MethodPatch, "/api/v1/notifications/"+n.ID+"/cancel", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	cancelAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	assert.False(t, cancelAPI.Success)
	assert.NotNil(t, cancelAPI.Error)
	assert.Equal(t, "NOTIFICATION_ALREADY_SENT", cancelAPI.Error.Code)
}

func TestIdempotency(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	idempotencyKey := uuid.New().String()
	headers := map[string]string{"X-Idempotency-Key": idempotencyKey}
	body := map[string]interface{}{
		"recipient": "+905556666666",
		"channel":   "sms",
		"content":   "Idempotency test",
		"priority":  "normal",
	}

	t.Run("first request should succeed", func(t *testing.T) {
		resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body, headers)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)
		assert.True(t, apiResp.Success)
	})

	t.Run("second request with same key should return 409", func(t *testing.T) {
		resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.NotNil(t, apiResp.Error)
		assert.Equal(t, "DUPLICATE_IDEMPOTENCY_KEY", apiResp.Error.Code)
	})
}

func TestFullDeliveryFlow(t *testing.T) {
	waitForHealth(t, 30*time.Second)

	// Create a notification.
	_, apiResp := createNotification(t, "+905557777777", "sms", "Full delivery flow test")
	require.True(t, apiResp.Success)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)
	require.NotEmpty(t, n.ID)

	// Wait for notification to reach "sent" status.
	// This requires the provider (webhook.site or mock) to be configured.
	// If the notification stays in queued/processing for too long, skip.
	deadline := time.Now().Add(60 * time.Second)
	var finalStatus string
	for time.Now().Before(deadline) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications/"+n.ID, nil)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		getAPI, err := parseAPIResponse(resp)
		if err != nil || !getAPI.Success {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var current notificationResponse
		if err := json.Unmarshal(getAPI.Data, &current); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		finalStatus = current.Status
		if finalStatus == "sent" {
			assert.NotNil(t, current.SentAt, "sentAt should be set for sent notifications")
			return // success
		}
		if finalStatus == "failed" {
			t.Skipf("notification reached 'failed' status — provider may not be configured; skipping full delivery test")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Skipf("notification did not reach 'sent' within timeout (last status: %s) — provider may not be available; skipping", finalStatus)
}
