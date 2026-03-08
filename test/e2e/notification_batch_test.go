//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBatchNotifications(t *testing.T) {
	body := map[string]interface{}{
		"notifications": []map[string]interface{}{
			{"recipient": "+905550000001", "channel": "sms", "content": "Batch msg 1", "priority": "normal"},
			{"recipient": "+905550000002", "channel": "email", "content": "Batch msg 2", "priority": "high"},
			{"recipient": "+905550000003", "channel": "push", "content": "Batch msg 3", "priority": "low"},
		},
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications/batch", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var batch batchResponse
	err = json.Unmarshal(apiResp.Data, &batch)
	require.NoError(t, err)

	assert.NotEmpty(t, batch.BatchID, "batchId should not be empty")
	assert.Len(t, batch.Notifications, 3, "batch should contain 3 notifications")

	for _, n := range batch.Notifications {
		assert.NotEmpty(t, n.ID)
		assert.Equal(t, batch.BatchID, *n.BatchID)
	}
}

func TestGetByBatchID(t *testing.T) {
	// Create a batch.
	body := map[string]interface{}{
		"notifications": []map[string]interface{}{
			{"recipient": "+905552222201", "channel": "sms", "content": "Batch get 1", "priority": "normal"},
			{"recipient": "+905552222202", "channel": "sms", "content": "Batch get 2", "priority": "normal"},
		},
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications/batch", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)

	var batch batchResponse
	err = json.Unmarshal(apiResp.Data, &batch)
	require.NoError(t, err)
	require.NotEmpty(t, batch.BatchID)

	// GET by batch ID.
	resp, err = makeRequest(http.MethodGet, "/api/v1/notifications/batch/"+batch.BatchID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	batchAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, batchAPI.Success)

	var items []notificationResponse
	err = json.Unmarshal(batchAPI.Data, &items)
	require.NoError(t, err)

	assert.Len(t, items, 2, "batch GET should return all 2 items")
	for _, item := range items {
		assert.Equal(t, batch.BatchID, *item.BatchID)
	}
}
