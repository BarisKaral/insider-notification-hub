//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelPendingNotification(t *testing.T) {
	// Create a notification.
	_, apiResp := createNotification(t, "+905554444444", "sms", "Cancel test")
	require.True(t, apiResp.Success)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	// Immediately cancel it. The notification could be pending, queued, failed, or already sent.
	resp, err := makeRequest(http.MethodPatch, "/api/v1/notifications/"+n.ID+"/cancel", nil)
	require.NoError(t, err)

	cancelAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)

	// If 409 with NOTIFICATION_ALREADY_SENT, skip — the consumer was too fast.
	if resp.StatusCode == http.StatusConflict {
		if cancelAPI.Error != nil && cancelAPI.Error.Code == "NOTIFICATION_ALREADY_SENT" {
			t.Skip("notification was already sent before cancel could execute — skipping race-sensitive test")
		}
	}

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
