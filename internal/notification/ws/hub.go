package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"

	"github.com/baris/notification-hub/pkg/logger"
)

// StatusBroadcaster broadcasts notification status changes (implemented by WebSocket hub).
type StatusBroadcaster interface {
	Broadcast(notificationID string, batchID *string, status string)
}

// Compile-time check: NotificationHub must implement StatusBroadcaster.
var _ StatusBroadcaster = (*NotificationHub)(nil)

// NotificationHub is the in-memory WebSocket hub for real-time notification status updates.
type NotificationHub struct {
	subscribers map[string]map[*websocket.Conn]bool
	mu          sync.RWMutex
}

// NotificationStatusUpdate represents a status change message sent over WebSocket.
type NotificationStatusUpdate struct {
	NotificationID string `json:"notificationId"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
}

// NewNotificationHub creates a new WebSocket hub.
func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		subscribers: make(map[string]map[*websocket.Conn]bool),
	}
}

// Subscribe adds a WebSocket connection to the subscriber set for a given key.
func (h *NotificationHub) Subscribe(key string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[key] == nil {
		h.subscribers[key] = make(map[*websocket.Conn]bool)
	}
	h.subscribers[key][conn] = true

	logger.Debug().
		Str("key", key).
		Msg("websocket client subscribed")
}

// Unsubscribe removes a WebSocket connection from the subscriber set for a given key.
func (h *NotificationHub) Unsubscribe(key string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.subscribers[key]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.subscribers, key)
		}
	}

	logger.Debug().
		Str("key", key).
		Msg("websocket client unsubscribed")
}

// Broadcast sends a status update to all subscribers of the notification ID and, if present, the batch ID.
func (h *NotificationHub) Broadcast(notificationID string, batchID *string, status string) {
	update := NotificationStatusUpdate{
		NotificationID: notificationID,
		Status:         status,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(update)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal websocket status update")
		return
	}

	h.broadcastToKey(notificationID, data)

	if batchID != nil {
		h.broadcastToKey(*batchID, data)
	}
}

// broadcastToKey sends a message to all subscribers of the given key.
func (h *NotificationHub) broadcastToKey(key string, data []byte) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.subscribers[key]))
	for conn := range h.subscribers[key] {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			logger.Error().Err(err).
				Str("key", key).
				Msg("failed to write websocket message, removing subscriber")
			h.Unsubscribe(key, conn)
		}
	}
}

// HandleNotificationWS is the Fiber WebSocket handler for /ws/notifications/:id.
func (h *NotificationHub) HandleNotificationWS(c *websocket.Conn) {
	key := c.Params("id")
	if key == "" {
		logger.Error().Msg("websocket connection missing notification id param")
		_ = c.Close()
		return
	}

	h.Subscribe(key, c)
	defer func() {
		h.Unsubscribe(key, c)
		_ = c.Close()
	}()

	// Read pump: keep connection alive and detect disconnect.
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

// HandleBatchWS is the Fiber WebSocket handler for /ws/notifications/batch/:batchId.
func (h *NotificationHub) HandleBatchWS(c *websocket.Conn) {
	key := c.Params("batchId")
	if key == "" {
		logger.Error().Msg("websocket connection missing batchId param")
		_ = c.Close()
		return
	}

	h.Subscribe(key, c)
	defer func() {
		h.Unsubscribe(key, c)
		_ = c.Close()
	}()

	// Read pump: keep connection alive and detect disconnect.
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}
