package provider

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrProviderConnectionFailed(t *testing.T) {
	assert.Equal(t, "PROVIDER_CONNECTION_FAILED", ErrProviderConnectionFailed.Code)
	assert.Equal(t, "failed to connect to notification provider", ErrProviderConnectionFailed.Message)
	assert.Equal(t, http.StatusBadGateway, ErrProviderConnectionFailed.StatusCode)
}

func TestErrProviderRejected(t *testing.T) {
	assert.Equal(t, "PROVIDER_REJECTED", ErrProviderRejected.Code)
	assert.Equal(t, "notification provider rejected the request", ErrProviderRejected.Message)
	assert.Equal(t, http.StatusBadGateway, ErrProviderRejected.StatusCode)
}

func TestErrProviderTimeout(t *testing.T) {
	assert.Equal(t, "PROVIDER_TIMEOUT", ErrProviderTimeout.Code)
	assert.Equal(t, "notification provider request timed out", ErrProviderTimeout.Message)
	assert.Equal(t, http.StatusGatewayTimeout, ErrProviderTimeout.StatusCode)
}

func TestErrProviderConnectionFailed_ImplementsErrorInterface(t *testing.T) {
	var err error = ErrProviderConnectionFailed
	assert.NotEmpty(t, err.Error())
	assert.Contains(t, err.Error(), "PROVIDER_CONNECTION_FAILED")
	assert.Contains(t, err.Error(), "failed to connect to notification provider")
}

func TestErrProviderRejected_ImplementsErrorInterface(t *testing.T) {
	var err error = ErrProviderRejected
	assert.NotEmpty(t, err.Error())
	assert.Contains(t, err.Error(), "PROVIDER_REJECTED")
	assert.Contains(t, err.Error(), "notification provider rejected the request")
}

func TestErrProviderTimeout_ImplementsErrorInterface(t *testing.T) {
	var err error = ErrProviderTimeout
	assert.NotEmpty(t, err.Error())
	assert.Contains(t, err.Error(), "PROVIDER_TIMEOUT")
	assert.Contains(t, err.Error(), "notification provider request timed out")
}

func TestErrProviderConnectionFailed_GetStatusCode(t *testing.T) {
	assert.Equal(t, http.StatusBadGateway, ErrProviderConnectionFailed.GetStatusCode())
}

func TestErrProviderRejected_GetStatusCode(t *testing.T) {
	assert.Equal(t, http.StatusBadGateway, ErrProviderRejected.GetStatusCode())
}

func TestErrProviderTimeout_GetStatusCode(t *testing.T) {
	assert.Equal(t, http.StatusGatewayTimeout, ErrProviderTimeout.GetStatusCode())
}

func TestProviderErrors_NilWrappedError(t *testing.T) {
	assert.Nil(t, ErrProviderConnectionFailed.Err)
	assert.Nil(t, ErrProviderRejected.Err)
	assert.Nil(t, ErrProviderTimeout.Err)
}

func TestProviderErrors_WithError(t *testing.T) {
	originalErr := assert.AnError
	wrapped := ErrProviderConnectionFailed.WithError(originalErr)

	assert.Equal(t, "PROVIDER_CONNECTION_FAILED", wrapped.Code)
	assert.Equal(t, "failed to connect to notification provider", wrapped.Message)
	assert.Equal(t, http.StatusBadGateway, wrapped.StatusCode)
	assert.Equal(t, originalErr, wrapped.Err)
	assert.Contains(t, wrapped.Error(), originalErr.Error())
}
