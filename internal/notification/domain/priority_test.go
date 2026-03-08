package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriority_ToUint8(t *testing.T) {
	tests := []struct {
		priority NotificationPriority
		expected uint8
	}{
		{NotificationPriorityHigh, 3},
		{NotificationPriorityNormal, 2},
		{NotificationPriorityLow, 1},
		{NotificationPriority("unknown"), 2},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.ToUint8())
		})
	}
}
