package domain

type NotificationPriority string

const (
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityLow    NotificationPriority = "low"
)

func (p NotificationPriority) ToUint8() uint8 {
	switch p {
	case NotificationPriorityHigh:
		return 3
	case NotificationPriorityNormal:
		return 2
	case NotificationPriorityLow:
		return 1
	default:
		return 2
	}
}
