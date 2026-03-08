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

func TestCreateNotification(t *testing.T) {
	resp, apiResp := createNotification(t, "+905551234567", "sms", "Hello from e2e test")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.True(t, apiResp.Success)
	require.Nil(t, apiResp.Error)

	var n notificationResponse
	err := json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	assert.NotEmpty(t, n.ID, "notification ID should not be empty")
	assert.Equal(t, "+905551234567", n.Recipient)
	assert.Equal(t, "sms", n.Channel)
	assert.Equal(t, "Hello from e2e test", n.Content)
	assert.Equal(t, "normal", n.Priority)
	assert.Contains(t, []string{"pending", "queued"}, n.Status, "initial status should be pending or queued")
	assert.NotEmpty(t, n.CreatedAt)
}

func TestCreateNotificationWithTemplate(t *testing.T) {
	// Step 1: Create a template.
	tmplName := fmt.Sprintf("e2e-template-%s", uuid.New().String()[:8])
	tmplResp, tmplAPI := createTemplate(t, tmplName, "sms", "Hello {{name}}, your code is {{code}}")

	require.Equal(t, http.StatusCreated, tmplResp.StatusCode)
	require.True(t, tmplAPI.Success)

	var tmpl templateResponse
	err := json.Unmarshal(tmplAPI.Data, &tmpl)
	require.NoError(t, err)
	require.NotEmpty(t, tmpl.ID)

	// Step 2: Create notification using template.
	body := map[string]interface{}{
		"recipient":  "+905559876543",
		"channel":    "sms",
		"templateId": tmpl.ID,
		"variables": map[string]string{
			"name": "Baris",
			"code": "1234",
		},
		"priority": "high",
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var n notificationResponse
	err = json.Unmarshal(apiResp.Data, &n)
	require.NoError(t, err)

	// Step 3: Verify rendered content.
	assert.Equal(t, "Hello Baris, your code is 1234", n.Content)
	assert.Equal(t, tmpl.ID, *n.TemplateID)
	assert.Equal(t, "high", n.Priority)
}
