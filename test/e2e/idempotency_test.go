//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotency(t *testing.T) {
	idempotencyKey := uuid.New().String()
	headers := map[string]string{"X-Idempotency-Key": idempotencyKey}
	body := map[string]interface{}{
		"recipient": "+905556666666",
		"channel":   "sms",
		"content":   "Idempotency test",
		"priority":  "normal",
	}

	t.Run("first request should succeed", func(t *testing.T) {
		resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body, headers)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)
		assert.True(t, apiResp.Success)
	})

	t.Run("second request with same key should return 409", func(t *testing.T) {
		resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.NotNil(t, apiResp.Error)
		assert.Equal(t, "DUPLICATE_IDEMPOTENCY_KEY", apiResp.Error.Code)
	})
}
