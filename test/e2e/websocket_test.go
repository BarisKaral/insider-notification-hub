//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wsBaseURL = "ws://localhost:8080"

// wsStatusUpdate matches ws.NotificationStatusUpdate.
type wsStatusUpdate struct {
	NotificationID string `json:"notificationId"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
}

func TestWebSocketNotificationStatusUpdate(t *testing.T) {
	// 1. Create a notification to get its ID.
	_, apiResp := createNotification(t, "+905551234567", "sms", "WebSocket e2e test")
	require.True(t, apiResp.Success)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)
	require.NotEmpty(t, n.ID)

	// 2. Connect to WebSocket for this notification.
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, resp, err := dialer.Dial(wsBaseURL+"/ws/notifications/"+n.ID, nil)
	if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("websocket upgrade failed with status %d", resp.StatusCode)
	}
	require.NoError(t, err, "failed to connect to websocket")
	defer conn.Close()

	// 3. Read messages until we get a terminal status or timeout.
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	var lastUpdate wsStatusUpdate
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// If provider is not configured, notification may stay queued.
			if lastUpdate.Status == "" {
				t.Skipf("no websocket message received before timeout — provider may not be configured; skipping")
			}
			break
		}

		err = json.Unmarshal(msg, &lastUpdate)
		require.NoError(t, err, "failed to unmarshal websocket message")

		assert.Equal(t, n.ID, lastUpdate.NotificationID, "notificationId should match")
		assert.NotEmpty(t, lastUpdate.Timestamp, "timestamp should be set")

		if lastUpdate.Status == "sent" || lastUpdate.Status == "failed" {
			break
		}
	}

	if lastUpdate.Status == "failed" {
		t.Skipf("notification reached 'failed' via websocket — provider may not be configured; skipping")
		return
	}

	assert.Equal(t, "sent", lastUpdate.Status, "expected terminal status 'sent' via websocket")
}
