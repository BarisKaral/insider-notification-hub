package notification

type NotificationStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusScheduled  NotificationStatus = "scheduled"
	StatusQueued     NotificationStatus = "queued"
	StatusProcessing NotificationStatus = "processing"
	StatusSent       NotificationStatus = "sent"
	StatusFailed     NotificationStatus = "failed"
	StatusRetrying   NotificationStatus = "retrying"
	StatusCancelled  NotificationStatus = "cancelled"
)

// CanTransitionTo checks if a status transition is valid per state machine.
func (s NotificationStatus) CanTransitionTo(target NotificationStatus) bool {
	transitions := map[NotificationStatus][]NotificationStatus{
		StatusPending:    {StatusQueued, StatusCancelled},
		StatusScheduled:  {StatusQueued, StatusCancelled},
		StatusQueued:     {StatusProcessing, StatusCancelled},
		StatusProcessing: {StatusSent, StatusFailed},
		StatusFailed:     {StatusRetrying, StatusCancelled},
		StatusRetrying:   {StatusProcessing},
	}
	for _, allowed := range transitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}
