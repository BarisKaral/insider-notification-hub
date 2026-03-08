//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNotificationByID(t *testing.T) {
	// Create a notification.
	_, apiResp := createNotification(t, "+905551111111", "sms", "Get by ID test")
	require.True(t, apiResp.Success)

	var created notificationResponse
	err := json.Unmarshal(apiResp.Data, &created)
	require.NoError(t, err)

	// GET by ID.
	resp, err := makeRequest(http.MethodGet, "/api/v1/notifications/"+created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, getAPI.Success)

	var fetched notificationResponse
	err = json.Unmarshal(getAPI.Data, &fetched)
	require.NoError(t, err)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, created.Recipient, fetched.Recipient)
	assert.Equal(t, created.Channel, fetched.Channel)
	assert.Equal(t, created.Content, fetched.Content)
	assert.Equal(t, created.Priority, fetched.Priority)
}
