package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppError(t *testing.T) {
	err := NewAppError("NOT_FOUND", "resource not found", 404)

	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "resource not found", err.Message)
	assert.Equal(t, 404, err.StatusCode)
	assert.Nil(t, err.Err)
}

func TestAppError_Error_WithoutWrapped(t *testing.T) {
	err := NewAppError("BAD_REQUEST", "invalid input", 400)
	assert.Equal(t, "BAD_REQUEST: invalid input", err.Error())
}

func TestAppError_Error_WithWrapped(t *testing.T) {
	inner := errors.New("db connection failed")
	err := NewAppError("DB_ERROR", "database error", 500).WithError(inner)
	assert.Equal(t, "DB_ERROR: database error: db connection failed", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("original")
	err := NewAppError("CODE", "msg", 500).WithError(inner)
	assert.Equal(t, inner, err.Unwrap())
}

func TestAppError_Unwrap_Nil(t *testing.T) {
	err := NewAppError("CODE", "msg", 500)
	assert.Nil(t, err.Unwrap())
}

func TestAppError_GetStatusCode(t *testing.T) {
	err := NewAppError("CODE", "msg", 422)
	assert.Equal(t, 422, err.GetStatusCode())
}

func TestAppError_WithError_ReturnsNewInstance(t *testing.T) {
	original := NewAppError("CODE", "msg", 500)
	inner := errors.New("wrapped")
	withErr := original.WithError(inner)

	assert.Nil(t, original.Err, "original should not be mutated")
	assert.Equal(t, inner, withErr.Err)
	assert.Equal(t, original.Code, withErr.Code)
	assert.Equal(t, original.Message, withErr.Message)
	assert.Equal(t, original.StatusCode, withErr.StatusCode)
}
