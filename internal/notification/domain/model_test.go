package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotification_TableName(t *testing.T) {
	n := Notification{}
	assert.Equal(t, "notifications", n.TableName())
}
