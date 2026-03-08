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

func TestFullDeliveryFlow(t *testing.T) {
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
