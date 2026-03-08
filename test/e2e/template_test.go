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

func TestCreateTemplate(t *testing.T) {
	name := fmt.Sprintf("e2e-create-%s", uuid.New().String()[:8])
	resp, apiResp := createTemplate(t, name, "sms", "Hello {{name}}")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.True(t, apiResp.Success)
	require.Nil(t, apiResp.Error)

	var tmpl templateResponse
	err := json.Unmarshal(apiResp.Data, &tmpl)
	require.NoError(t, err)

	assert.NotEmpty(t, tmpl.ID, "template ID should not be empty")
	assert.Equal(t, name, tmpl.Name)
	assert.Equal(t, "sms", tmpl.Channel)
	assert.Equal(t, "Hello {{name}}", tmpl.Content)
	assert.NotEmpty(t, tmpl.CreatedAt)
	assert.NotEmpty(t, tmpl.UpdatedAt)
}

func TestGetTemplateByID(t *testing.T) {
	// Create a template first.
	name := fmt.Sprintf("e2e-get-%s", uuid.New().String()[:8])
	_, apiResp := createTemplate(t, name, "email", "Dear {{user}}, welcome!")
	require.True(t, apiResp.Success)

	var created templateResponse
	err := json.Unmarshal(apiResp.Data, &created)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	// GET the template by ID.
	resp, err := makeRequest(http.MethodGet, "/api/v1/templates/"+created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, getAPI.Success)

	var fetched templateResponse
	err = json.Unmarshal(getAPI.Data, &fetched)
	require.NoError(t, err)

	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, name, fetched.Name)
	assert.Equal(t, "email", fetched.Channel)
	assert.Equal(t, "Dear {{user}}, welcome!", fetched.Content)
	assert.NotEmpty(t, fetched.CreatedAt)
	assert.NotEmpty(t, fetched.UpdatedAt)
}

func TestListTemplates(t *testing.T) {
	// Create a couple of templates to ensure the list is not empty.
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("e2e-list-%s-%d", uuid.New().String()[:8], i)
		_, apiResp := createTemplate(t, name, "sms", fmt.Sprintf("List test %d", i))
		require.True(t, apiResp.Success)
	}

	resp, err := makeRequest(http.MethodGet, "/api/v1/templates?limit=20&offset=0", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, apiResp.Success)

	var paginated templatePaginatedResponse
	err = json.Unmarshal(apiResp.Data, &paginated)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(paginated.Items), 2, "should have at least 2 items")
	assert.GreaterOrEqual(t, paginated.Total, int64(2))
	assert.Equal(t, 20, paginated.Limit)
	assert.Equal(t, 0, paginated.Offset)
}

func TestUpdateTemplate(t *testing.T) {
	// Create a template.
	name := fmt.Sprintf("e2e-update-%s", uuid.New().String()[:8])
	_, apiResp := createTemplate(t, name, "sms", "Original content")
	require.True(t, apiResp.Success)

	var created templateResponse
	err := json.Unmarshal(apiResp.Data, &created)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	// Update the template.
	updatedName := fmt.Sprintf("e2e-updated-%s", uuid.New().String()[:8])
	updateBody := map[string]interface{}{
		"name":    updatedName,
		"channel": "email",
		"content": "Updated content for {{user}}",
	}
	resp, err := makeRequest(http.MethodPut, "/api/v1/templates/"+created.ID, updateBody)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	updateAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, updateAPI.Success)

	var updated templateResponse
	err = json.Unmarshal(updateAPI.Data, &updated)
	require.NoError(t, err)

	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, updatedName, updated.Name)
	assert.Equal(t, "email", updated.Channel)
	assert.Equal(t, "Updated content for {{user}}", updated.Content)

	// GET again to verify changes persisted.
	resp, err = makeRequest(http.MethodGet, "/api/v1/templates/"+created.ID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	getAPI, err := parseAPIResponse(resp)
	require.NoError(t, err)
	require.True(t, getAPI.Success)

	var fetched templateResponse
	err = json.Unmarshal(getAPI.Data, &fetched)
	require.NoError(t, err)

	assert.Equal(t, updatedName, fetched.Name)
	assert.Equal(t, "email", fetched.Channel)
	assert.Equal(t, "Updated content for {{user}}", fetched.Content)
}

func TestDeleteTemplate(t *testing.T) {
	// Create a template.
	name := fmt.Sprintf("e2e-delete-%s", uuid.New().String()[:8])
	_, apiResp := createTemplate(t, name, "push", "Delete me")
	require.True(t, apiResp.Success)

	var created templateResponse
	err := json.Unmarshal(apiResp.Data, &created)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	// DELETE the template.
	resp, err := makeRequest(http.MethodDelete, "/api/v1/templates/"+created.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// GET again — should return 404.
	resp, err = makeRequest(http.MethodGet, "/api/v1/templates/"+created.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
