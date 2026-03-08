//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListNotifications(t *testing.T) {
	// Create a few notifications to ensure the list is not empty.
	for i := 0; i < 3; i++ {
		createNotification(t, fmt.Sprintf("+90555300000%d", i), "sms", fmt.Sprintf("List test %d", i))
	}

	resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?limit=20&offset=0", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var paginated paginatedResponse
	err = json.Unmarshal(apiResp.Data, &paginated)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(paginated.Items), 3, "should have at least 3 items")
	assert.GreaterOrEqual(t, paginated.Total, int64(3))
	assert.Equal(t, 20, paginated.Limit)
	assert.Equal(t, 0, paginated.Offset)
}

func TestListWithFilters(t *testing.T) {
	// Create one SMS and one email notification with unique recipients.
	tag := uuid.New().String()[:8]
	smsRecipient := fmt.Sprintf("+90555filter-sms-%s", tag)
	emailRecipient := fmt.Sprintf("filter-email-%s@test.com", tag)

	createNotification(t, smsRecipient, "sms", "Filter test SMS")
	createNotification(t, emailRecipient, "email", "Filter test Email")

	t.Run("filter by channel=sms", func(t *testing.T) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?channel=sms&limit=100", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)

		var paginated paginatedResponse
		err = json.Unmarshal(apiResp.Data, &paginated)
		require.NoError(t, err)

		for _, item := range paginated.Items {
			assert.Equal(t, "sms", item.Channel, "filtered results should only contain sms channel")
		}
	})

	t.Run("filter by channel=email", func(t *testing.T) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications?channel=email&limit=100", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		apiResp, err := parseAPIResponse(resp)
		require.NoError(t, err)

		var paginated paginatedResponse
		err = json.Unmarshal(apiResp.Data, &paginated)
		require.NoError(t, err)

		for _, item := range paginated.Items {
			assert.Equal(t, "email", item.Channel, "filtered results should only contain email channel")
		}
	})
}
