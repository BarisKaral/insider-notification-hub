package domain

type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "pending"
	NotificationStatusScheduled  NotificationStatus = "scheduled"
	NotificationStatusQueued     NotificationStatus = "queued"
	NotificationStatusProcessing NotificationStatus = "processing"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusFailed     NotificationStatus = "failed"
	NotificationStatusRetrying   NotificationStatus = "retrying"
	NotificationStatusCancelled  NotificationStatus = "cancelled"
)

// CanTransitionTo checks if a status transition is valid per state machine.
func (s NotificationStatus) CanTransitionTo(target NotificationStatus) bool {
	transitions := map[NotificationStatus][]NotificationStatus{
		NotificationStatusPending:    {NotificationStatusQueued, NotificationStatusCancelled},
		NotificationStatusScheduled:  {NotificationStatusQueued, NotificationStatusCancelled},
		NotificationStatusQueued:     {NotificationStatusProcessing, NotificationStatusCancelled},
		NotificationStatusProcessing: {NotificationStatusSent, NotificationStatusFailed},
		NotificationStatusFailed:     {NotificationStatusRetrying, NotificationStatusCancelled},
		NotificationStatusRetrying:   {NotificationStatusProcessing},
	}
	for _, allowed := range transitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}
