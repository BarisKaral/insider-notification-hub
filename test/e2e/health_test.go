//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	resp, err := makeRequest(http.MethodGet, "/health", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := readBody(resp)
	require.NoError(t, err)

	var health healthResponse
	err = json.Unmarshal(body, &health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "up", health.Checks["database"])
	assert.Equal(t, "up", health.Checks["rabbitmq"])
}
