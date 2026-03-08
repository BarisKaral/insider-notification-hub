package ws

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bariskaral/insider-notification-hub/pkg/logger"
)

func init() {
	logger.Init("debug")
}

// startTestServer creates a Fiber app with WebSocket routes, starts it on a random
// port, and returns the app along with a base URL like "ws://127.0.0.1:PORT".
func startTestServer(t *testing.T, hub *NotificationHub) (*fiber.App, string) {
	t.Helper()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/notifications/:id", websocket.New(hub.HandleNotificationWS))
	app.Get("/ws/notifications/batch/:batchId", websocket.New(hub.HandleBatchWS))

	// Routes for testing the empty-param code paths in handlers.
	// Because Fiber's /:id pattern requires a non-empty segment, these routes
	// exercise HandleNotificationWS / HandleBatchWS with Params returning "".
	app.Get("/ws/empty-notif", websocket.New(hub.HandleNotificationWS))
	app.Get("/ws/empty-batch", websocket.New(hub.HandleBatchWS))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = app.Listener(ln)
	}()

	addr := fmt.Sprintf("ws://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
	// Brief pause so the server is ready to accept connections.
	time.Sleep(50 * time.Millisecond)
	return app, addr
}

// dialWS is a helper that opens a gorilla websocket client connection.
func dialWS(t *testing.T, url string) *gorilla.Conn {
	t.Helper()
	dialer := gorilla.Dialer{
		HandshakeTimeout: 2 * time.Second,
	}
	conn, resp, err := dialer.Dial(url, http.Header{})
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	return conn
}

// readStatusUpdate reads a single JSON message from the gorilla WS connection
// and decodes it into a NotificationStatusUpdate.
func readStatusUpdate(t *testing.T, conn *gorilla.Conn) NotificationStatusUpdate {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var update NotificationStatusUpdate
	require.NoError(t, json.Unmarshal(msg, &update))
	return update
}

// ---------------------------------------------------------------------------
// Existing unit tests (no server needed)
// ---------------------------------------------------------------------------

func TestNewNotificationHub_ReturnsNonNil(t *testing.T) {
	hub := NewNotificationHub()

	assert.NotNil(t, hub)
}

func TestNewNotificationHub_InitializesSubscribersMap(t *testing.T) {
	hub := NewNotificationHub()

	assert.NotNil(t, hub.subscribers)
	assert.Empty(t, hub.subscribers)
}

func TestNewNotificationHub_SubscribersMapIsUsable(t *testing.T) {
	hub := NewNotificationHub()

	_, exists := hub.subscribers["nonexistent"]
	assert.False(t, exists)
}

func TestNotificationHub_ImplementsStatusBroadcaster(t *testing.T) {
	var _ StatusBroadcaster = (*NotificationHub)(nil)
}

func TestNotificationHub_Broadcast_WithNilBatchID_DoesNotPanic(t *testing.T) {
	hub := NewNotificationHub()

	assert.NotPanics(t, func() {
		hub.Broadcast("notif-123", nil, "delivered")
	})
}

func TestNotificationHub_Broadcast_WithBatchID_DoesNotPanic(t *testing.T) {
	hub := NewNotificationHub()
	batchID := "batch-456"

	assert.NotPanics(t, func() {
		hub.Broadcast("notif-123", &batchID, "delivered")
	})
}

func TestNotificationHub_Broadcast_NoSubscribers_DoesNotPanic(t *testing.T) {
	hub := NewNotificationHub()

	assert.NotPanics(t, func() {
		hub.Broadcast("nonexistent-id", nil, "sent")
	})
}

func TestNotificationStatusUpdate_JSONMarshal(t *testing.T) {
	update := NotificationStatusUpdate{
		NotificationID: "notif-789",
		Status:         "delivered",
		Timestamp:      "2026-03-08T10:00:00Z",
	}

	data, err := json.Marshal(update)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "notif-789", result["notificationId"])
	assert.Equal(t, "delivered", result["status"])
	assert.Equal(t, "2026-03-08T10:00:00Z", result["timestamp"])
}

func TestNotificationStatusUpdate_JSONUnmarshal(t *testing.T) {
	jsonStr := `{"notificationId":"notif-abc","status":"failed","timestamp":"2026-03-08T12:00:00Z"}`

	var update NotificationStatusUpdate
	err := json.Unmarshal([]byte(jsonStr), &update)
	require.NoError(t, err)

	assert.Equal(t, "notif-abc", update.NotificationID)
	assert.Equal(t, "failed", update.Status)
	assert.Equal(t, "2026-03-08T12:00:00Z", update.Timestamp)
}

func TestNotificationStatusUpdate_JSONRoundTrip(t *testing.T) {
	original := NotificationStatusUpdate{
		NotificationID: "round-trip-1",
		Status:         "sent",
		Timestamp:      "2026-03-08T15:30:00Z",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded NotificationStatusUpdate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestNotificationStatusUpdate_JSONFieldNames(t *testing.T) {
	update := NotificationStatusUpdate{
		NotificationID: "id-1",
		Status:         "pending",
		Timestamp:      "2026-03-08T10:00:00Z",
	}

	data, err := json.Marshal(update)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"notificationId"`)
	assert.Contains(t, jsonStr, `"status"`)
	assert.Contains(t, jsonStr, `"timestamp"`)
	assert.NotContains(t, jsonStr, `"NotificationID"`)
	assert.NotContains(t, jsonStr, `"Status"`)
	assert.NotContains(t, jsonStr, `"Timestamp"`)
}

func TestNotificationStatusUpdate_EmptyFields(t *testing.T) {
	update := NotificationStatusUpdate{}

	data, err := json.Marshal(update)
	require.NoError(t, err)

	var decoded NotificationStatusUpdate
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "", decoded.NotificationID)
	assert.Equal(t, "", decoded.Status)
	assert.Equal(t, "", decoded.Timestamp)
}

func TestStatusBroadcaster_InterfaceMethod(t *testing.T) {
	hub := NewNotificationHub()
	var broadcaster StatusBroadcaster = hub

	assert.NotPanics(t, func() {
		broadcaster.Broadcast("test-id", nil, "delivered")
	})
}

func TestNotificationHub_ConcurrentBroadcast_DoesNotPanic(t *testing.T) {
	hub := NewNotificationHub()
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			batchID := "batch-1"
			hub.Broadcast("notif-concurrent", &batchID, "sent")
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// ---------------------------------------------------------------------------
// Integration tests using a real Fiber WebSocket server
// ---------------------------------------------------------------------------

func TestHandleNotificationWS_SubscribesAndReceivesBroadcast(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/notif-100")
	defer conn.Close()
	// Allow time for the handler to call Subscribe.
	time.Sleep(100 * time.Millisecond)

	// Broadcast a status update for the subscribed notification.
	hub.Broadcast("notif-100", nil, "sent")

	update := readStatusUpdate(t, conn)
	assert.Equal(t, "notif-100", update.NotificationID)
	assert.Equal(t, "sent", update.Status)
	assert.NotEmpty(t, update.Timestamp)
}

func TestHandleBatchWS_SubscribesAndReceivesBroadcast(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/batch/batch-200")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	batchID := "batch-200"
	hub.Broadcast("some-notif", &batchID, "delivered")

	update := readStatusUpdate(t, conn)
	assert.Equal(t, "some-notif", update.NotificationID)
	assert.Equal(t, "delivered", update.Status)
	assert.NotEmpty(t, update.Timestamp)
}

func TestHandleNotificationWS_UnsubscribesOnClientClose(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/notif-300")
	time.Sleep(100 * time.Millisecond)

	// Verify the subscriber is registered.
	hub.mutex.RLock()
	assert.Len(t, hub.subscribers["notif-300"], 1)
	hub.mutex.RUnlock()

	// Close the client connection.
	conn.Close()
	// Allow time for the server to detect the disconnect and run Unsubscribe.
	time.Sleep(200 * time.Millisecond)

	// The subscriber map entry should be cleaned up.
	hub.mutex.RLock()
	_, exists := hub.subscribers["notif-300"]
	hub.mutex.RUnlock()
	assert.False(t, exists)
}

func TestHandleBatchWS_UnsubscribesOnClientClose(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/batch/batch-300")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	assert.Len(t, hub.subscribers["batch-300"], 1)
	hub.mutex.RUnlock()

	conn.Close()
	time.Sleep(200 * time.Millisecond)

	hub.mutex.RLock()
	_, exists := hub.subscribers["batch-300"]
	hub.mutex.RUnlock()
	assert.False(t, exists)
}

func TestBroadcast_WithBatchID_ReachesBothSubscribers(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// One client subscribes to the notification ID.
	notifConn := dialWS(t, addr+"/ws/notifications/notif-400")
	defer notifConn.Close()
	// Another client subscribes to the batch ID.
	batchConn := dialWS(t, addr+"/ws/notifications/batch/batch-400")
	defer batchConn.Close()
	time.Sleep(100 * time.Millisecond)

	batchID := "batch-400"
	hub.Broadcast("notif-400", &batchID, "failed")

	// Both clients should receive the update.
	notifUpdate := readStatusUpdate(t, notifConn)
	assert.Equal(t, "notif-400", notifUpdate.NotificationID)
	assert.Equal(t, "failed", notifUpdate.Status)

	batchUpdate := readStatusUpdate(t, batchConn)
	assert.Equal(t, "notif-400", batchUpdate.NotificationID)
	assert.Equal(t, "failed", batchUpdate.Status)
}

func TestBroadcast_MultipleSubscribersOnSameKey(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn1 := dialWS(t, addr+"/ws/notifications/notif-500")
	defer conn1.Close()
	conn2 := dialWS(t, addr+"/ws/notifications/notif-500")
	defer conn2.Close()
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	assert.Len(t, hub.subscribers["notif-500"], 2)
	hub.mutex.RUnlock()

	hub.Broadcast("notif-500", nil, "sent")

	update1 := readStatusUpdate(t, conn1)
	update2 := readStatusUpdate(t, conn2)

	assert.Equal(t, "sent", update1.Status)
	assert.Equal(t, "sent", update2.Status)
}

func TestBroadcast_WriteFailed_RemovesSubscriber(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/notif-600")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	assert.Len(t, hub.subscribers["notif-600"], 1)
	hub.mutex.RUnlock()

	// Close the underlying TCP connection without a proper WebSocket close.
	// This will cause WriteMessage to fail on the server side.
	conn.UnderlyingConn().Close()
	time.Sleep(100 * time.Millisecond)

	// Broadcast after the client's TCP is dead — WriteMessage should fail
	// and broadcastToKey should remove the subscriber via Unsubscribe.
	hub.Broadcast("notif-600", nil, "sent")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	_, exists := hub.subscribers["notif-600"]
	hub.mutex.RUnlock()
	assert.False(t, exists, "subscriber should be removed after write failure")
}

func TestSubscribe_CreatesNewKeyEntry(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/brand-new-key")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	conns, exists := hub.subscribers["brand-new-key"]
	hub.mutex.RUnlock()

	assert.True(t, exists)
	assert.Len(t, conns, 1)
}

func TestUnsubscribe_RemovesKeyWhenLastConnectionGone(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/single-sub")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	assert.Contains(t, hub.subscribers, "single-sub")
	hub.mutex.RUnlock()

	conn.Close()
	time.Sleep(200 * time.Millisecond)

	hub.mutex.RLock()
	_, exists := hub.subscribers["single-sub"]
	hub.mutex.RUnlock()
	assert.False(t, exists, "key should be removed when last subscriber disconnects")
}

func TestUnsubscribe_KeepsKeyWhenOtherConnectionsRemain(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn1 := dialWS(t, addr+"/ws/notifications/multi-sub")
	conn2 := dialWS(t, addr+"/ws/notifications/multi-sub")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	assert.Len(t, hub.subscribers["multi-sub"], 2)
	hub.mutex.RUnlock()

	// Close only one connection.
	conn1.Close()
	time.Sleep(200 * time.Millisecond)

	hub.mutex.RLock()
	conns, exists := hub.subscribers["multi-sub"]
	hub.mutex.RUnlock()
	assert.True(t, exists, "key should still exist with one subscriber remaining")
	assert.Len(t, conns, 1)

	conn2.Close()
	time.Sleep(200 * time.Millisecond)

	hub.mutex.RLock()
	_, exists = hub.subscribers["multi-sub"]
	hub.mutex.RUnlock()
	assert.False(t, exists, "key should be removed when all subscribers gone")
}

func TestConcurrentSubscribeUnsubscribeBroadcast(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	var wg sync.WaitGroup
	const concurrency = 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := fmt.Sprintf("concurrent-%d", idx)
			conn := dialWS(t, addr+"/ws/notifications/"+key)
			time.Sleep(50 * time.Millisecond)

			hub.Broadcast(key, nil, "sent")

			// Read the broadcast message.
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, msg, err := conn.ReadMessage()
			if err == nil {
				var update NotificationStatusUpdate
				_ = json.Unmarshal(msg, &update)
				assert.Equal(t, key, update.NotificationID)
			}

			conn.Close()
		}(i)
	}

	wg.Wait()
}

func TestBroadcast_MultipleBroadcasts_AllReceived(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/notif-multi")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	statuses := []string{"pending", "sent", "delivered"}
	for _, s := range statuses {
		hub.Broadcast("notif-multi", nil, s)
	}

	for _, expected := range statuses {
		update := readStatusUpdate(t, conn)
		assert.Equal(t, "notif-multi", update.NotificationID)
		assert.Equal(t, expected, update.Status)
	}
}

func TestBroadcast_NotificationAndBatchRouting(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// Subscribe to notification key only.
	notifConn := dialWS(t, addr+"/ws/notifications/notif-700")
	defer notifConn.Close()
	time.Sleep(100 * time.Millisecond)

	// Broadcast without batchID — only notif subscriber should get it.
	hub.Broadcast("notif-700", nil, "sent")

	update := readStatusUpdate(t, notifConn)
	assert.Equal(t, "notif-700", update.NotificationID)
	assert.Equal(t, "sent", update.Status)
}

func TestUnsubscribe_NonexistentKey_DoesNotPanic(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// Connect and subscribe, then manually unsubscribe a nonexistent key.
	conn := dialWS(t, addr+"/ws/notifications/notif-800")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe a key that was never subscribed. This exercises the
	// "key not found" branch in Unsubscribe. We need a real *websocket.Conn
	// to call Unsubscribe, but we can get it indirectly: the hub already has
	// one under "notif-800". We'll try to unsubscribe a completely different key.
	// Since Unsubscribe checks for key existence, it should just be a no-op.
	hub.mutex.RLock()
	var existingConn *websocket.Conn
	for c := range hub.subscribers["notif-800"] {
		existingConn = c
		break
	}
	hub.mutex.RUnlock()
	require.NotNil(t, existingConn)

	assert.NotPanics(t, func() {
		hub.Unsubscribe("does-not-exist", existingConn)
	})
}

func TestBroadcast_TimestampIsRFC3339(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/notif-ts")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	before := time.Now().UTC()
	hub.Broadcast("notif-ts", nil, "sent")
	after := time.Now().UTC()

	update := readStatusUpdate(t, conn)

	ts, err := time.Parse(time.RFC3339, update.Timestamp)
	require.NoError(t, err)
	assert.False(t, ts.Before(before.Truncate(time.Second)))
	assert.False(t, ts.After(after.Add(time.Second)))
}

func TestBroadcast_StatusUpdateContainsCorrectFields(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	conn := dialWS(t, addr+"/ws/notifications/field-check")
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	hub.Broadcast("field-check", nil, "delivered")

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, rawMsg, err := conn.ReadMessage()
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(rawMsg, &raw))

	assert.Equal(t, "field-check", raw["notificationId"])
	assert.Equal(t, "delivered", raw["status"])
	assert.Contains(t, raw, "timestamp")
	// Should only have these 3 keys.
	assert.Len(t, raw, 3)
}

func TestHandleNotificationWS_EmptyParam_ClosesConnection(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// Connect to the route that has no :id param, so c.Params("id") returns "".
	conn := dialWS(t, addr+"/ws/empty-notif")
	defer conn.Close()

	// The handler should close the connection immediately. Reading should fail.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "connection should be closed by handler when param is empty")
}

func TestHandleBatchWS_EmptyParam_ClosesConnection(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// Connect to the route that has no :batchId param, so c.Params("batchId") returns "".
	conn := dialWS(t, addr+"/ws/empty-batch")
	defer conn.Close()

	// The handler should close the connection immediately. Reading should fail.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "connection should be closed by handler when param is empty")
}

func TestBroadcastToKey_WriteError_RemovesSubscriber(t *testing.T) {
	hub := NewNotificationHub()
	app, addr := startTestServer(t, hub)
	defer app.Shutdown()

	// Connect a client. The handler subscribes it to the key.
	conn := dialWS(t, addr+"/ws/notifications/write-err")
	time.Sleep(100 * time.Millisecond)

	// Verify subscriber exists.
	hub.mutex.RLock()
	require.Len(t, hub.subscribers["write-err"], 1)

	// Grab a reference to the server-side *websocket.Conn for that key.
	var serverConn *websocket.Conn
	for c := range hub.subscribers["write-err"] {
		serverConn = c
		break
	}
	hub.mutex.RUnlock()
	require.NotNil(t, serverConn)

	// Close the server-side underlying connection to force a write error
	// on the next WriteMessage call.
	_ = serverConn.Conn.Close()
	// Also close our client to stop the read pump from holding the entry.
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	// Re-subscribe the now-broken connection manually so we can trigger
	// the write-error branch in broadcastToKey. The read pump already
	// unsubscribed it, so we add it back.
	hub.Subscribe("write-err", serverConn)

	hub.mutex.RLock()
	require.Len(t, hub.subscribers["write-err"], 1)
	hub.mutex.RUnlock()

	// Broadcasting should hit the WriteMessage error path and auto-unsubscribe.
	hub.Broadcast("write-err", nil, "sent")
	time.Sleep(100 * time.Millisecond)

	hub.mutex.RLock()
	_, exists := hub.subscribers["write-err"]
	hub.mutex.RUnlock()
	assert.False(t, exists, "subscriber should be removed after WriteMessage error")
}
