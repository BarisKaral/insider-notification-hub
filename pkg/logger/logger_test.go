package logger

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInit_WithDebugLevel(t *testing.T) {
	Init("debug")
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestInit_WithInfoLevel(t *testing.T) {
	Init("info")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInit_WithErrorLevel(t *testing.T) {
	Init("error")
	assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())
}

func TestInit_WithWarnLevel(t *testing.T) {
	Init("warn")
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
}

func TestInit_WithInvalidLevel_FallsBackToInfo(t *testing.T) {
	Init("not-a-real-level")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInfo_ReturnsNonNilEvent(t *testing.T) {
	Init("debug")
	event := Info()
	assert.NotNil(t, event)
}

func TestError_ReturnsNonNilEvent(t *testing.T) {
	Init("debug")
	event := Error()
	assert.NotNil(t, event)
}

func TestDebug_ReturnsNonNilEvent(t *testing.T) {
	Init("debug")
	event := Debug()
	assert.NotNil(t, event)
}

func TestFatal_ReturnsNonNilEvent(t *testing.T) {
	Init("debug")
	event := Fatal()
	assert.NotNil(t, event)
}

func TestWithCorrelationID_ReturnsLogger(t *testing.T) {
	Init("info")
	l := WithCorrelationID("test-correlation-id")
	assert.NotNil(t, l)
}

func TestWithCorrelationID_DifferentIDs(t *testing.T) {
	Init("info")
	l1 := WithCorrelationID("id-1")
	l2 := WithCorrelationID("id-2")
	assert.NotNil(t, l1)
	assert.NotNil(t, l2)
}

func TestInit_CalledMultipleTimes_ChangesLevel(t *testing.T) {
	Init("debug")
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	Init("error")
	assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())

	Init("info")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestDebug_DisabledAtInfoLevel(t *testing.T) {
	Init("info")
	// Debug events are disabled when global level is info,
	// zerolog returns nil for disabled events
	event := Debug()
	assert.Nil(t, event)
}

func TestInfo_EnabledAtInfoLevel(t *testing.T) {
	Init("info")
	event := Info()
	assert.NotNil(t, event)
}

func TestError_EnabledAtInfoLevel(t *testing.T) {
	Init("info")
	event := Error()
	assert.NotNil(t, event)
}
