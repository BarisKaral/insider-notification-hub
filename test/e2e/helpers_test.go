//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestMain waits for all services to be ready before running any tests.
func TestMain(m *testing.M) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := makeRequest(http.MethodGet, "/health", nil)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	code := m.Run()
	os.Exit(code)
}

const baseURL = "http://localhost:8080"

// apiResponse is the generic API envelope used by the server.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *errorInfo      `json:"error,omitempty"`
}

type errorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// healthResponse matches pkg/health.HealthResponse.
type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// notificationResponse matches the API DTO.
type notificationResponse struct {
	ID            string  `json:"id"`
	Recipient     string  `json:"recipient"`
	Channel       string  `json:"channel"`
	Content       string  `json:"content"`
	Priority      string  `json:"priority"`
	Status        string  `json:"status"`
	BatchID       *string `json:"batchId,omitempty"`
	TemplateID    *string `json:"templateId,omitempty"`
	RetryCount    int     `json:"retryCount"`
	ScheduledAt   *string `json:"scheduledAt,omitempty"`
	SentAt        *string `json:"sentAt,omitempty"`
	FailedAt      *string `json:"failedAt,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

// batchResponse matches NotificationBatchResponse.
type batchResponse struct {
	BatchID       string                 `json:"batchId"`
	Notifications []notificationResponse `json:"notifications"`
}

// paginatedResponse matches NotificationPaginatedResponse.
type paginatedResponse struct {
	Items  []notificationResponse `json:"items"`
	Total  int64                  `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// templateResponse matches template.TemplateResponse.
type templateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// templatePaginatedResponse matches the template list endpoint response.
type templatePaginatedResponse struct {
	Items  []templateResponse `json:"items"`
	Total  int64              `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

// makeRequest sends an HTTP request and returns the response.
// body can be nil for requests without a payload.
func makeRequest(method, path string, body interface{}, headers ...map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply optional headers.
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// readBody reads and closes the response body, returning the raw bytes.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// parseAPIResponse reads the response body and decodes the API envelope.
func parseAPIResponse(resp *http.Response) (*apiResponse, error) {
	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(body))
	}
	return &result, nil
}

// waitForHealth polls the health endpoint until it returns 200 or the timeout expires.
func waitForHealth(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := makeRequest(http.MethodGet, "/health", nil)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.Fail(t, "health check did not return 200 within timeout")
}

// waitForStatus polls a notification by ID until its status matches the expected value or timeout.
func waitForStatus(t *testing.T, id, expectedStatus string, timeout time.Duration) notificationResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := makeRequest(http.MethodGet, "/api/v1/notifications/"+id, nil)
		if err == nil && resp.StatusCode == http.StatusOK {
			apiResp, err := parseAPIResponse(resp)
			if err == nil && apiResp.Success {
				var n notificationResponse
				if err := json.Unmarshal(apiResp.Data, &n); err == nil {
					if n.Status == expectedStatus {
						return n
					}
				}
			}
		} else if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.Failf(t, "timeout waiting for status", "notification %s did not reach status %q within %v", id, expectedStatus, timeout)
	return notificationResponse{} // unreachable
}

// createNotification is a shorthand to POST a single notification.
func createNotification(t *testing.T, recipient, channel, content string) (*http.Response, *apiResponse) {
	t.Helper()
	body := map[string]interface{}{
		"recipient": recipient,
		"channel":   channel,
		"content":   content,
		"priority":  "normal",
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/notifications", body)
	require.NoError(t, err, "failed to send create notification request")
	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err, "failed to parse create notification response")
	return resp, apiResp
}

// createTemplate is a shorthand to POST a new template.
func createTemplate(t *testing.T, name, channel, content string) (*http.Response, *apiResponse) {
	t.Helper()
	body := map[string]interface{}{
		"name":    name,
		"channel": channel,
		"content": content,
	}
	resp, err := makeRequest(http.MethodPost, "/api/v1/templates", body)
	require.NoError(t, err, "failed to send create template request")
	apiResp, err := parseAPIResponse(resp)
	require.NoError(t, err, "failed to parse create template response")
	return resp, apiResp
}
