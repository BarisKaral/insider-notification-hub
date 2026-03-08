package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransitionTo_ValidTransitions(t *testing.T) {
	valid := []struct {
		from, to NotificationStatus
	}{
		{NotificationStatusPending, NotificationStatusQueued},
		{NotificationStatusPending, NotificationStatusCancelled},
		{NotificationStatusScheduled, NotificationStatusQueued},
		{NotificationStatusScheduled, NotificationStatusCancelled},
		{NotificationStatusQueued, NotificationStatusProcessing},
		{NotificationStatusQueued, NotificationStatusCancelled},
		{NotificationStatusProcessing, NotificationStatusSent},
		{NotificationStatusProcessing, NotificationStatusFailed},
		{NotificationStatusFailed, NotificationStatusRetrying},
		{NotificationStatusFailed, NotificationStatusCancelled},
		{NotificationStatusRetrying, NotificationStatusProcessing},
	}

	for _, tt := range valid {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			assert.True(t, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestCanTransitionTo_InvalidTransitions(t *testing.T) {
	invalid := []struct {
		from, to NotificationStatus
	}{
		{NotificationStatusSent, NotificationStatusFailed},
		{NotificationStatusSent, NotificationStatusCancelled},
		{NotificationStatusCancelled, NotificationStatusPending},
		{NotificationStatusPending, NotificationStatusSent},
		{NotificationStatusQueued, NotificationStatusFailed},
		{NotificationStatusProcessing, NotificationStatusCancelled},
		{NotificationStatusRetrying, NotificationStatusSent},
	}

	for _, tt := range invalid {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			assert.False(t, tt.from.CanTransitionTo(tt.to))
		})
	}
}
