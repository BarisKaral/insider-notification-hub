package ws

// NotificationStatusUpdate represents a status change message sent over WebSocket.
type NotificationStatusUpdate struct {
	NotificationID string `json:"notificationId"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
}
