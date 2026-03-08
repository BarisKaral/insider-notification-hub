package email

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bariskaral/insider-notification-hub/internal/notification/provider"
)

func newTestEmailProvider(serverURL string) *EmailProvider {
	config := EmailClientConfig{
		URL:        serverURL,
		AuthKey:    "test-auth-key",
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	return NewEmailProvider(config)
}

func TestEmailProvider_Send_SuccessWithValidJSON(t *testing.T) {
	expectedResponse := provider.ProviderResponse{
		MessageID: "msg-12345",
		Status:    "delivered",
		Timestamp: "2026-03-08T10:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "msg-12345", resp.MessageID)
	assert.Equal(t, "delivered", resp.Status)
	assert.Equal(t, "2026-03-08T10:00:00Z", resp.Timestamp)
}

func TestEmailProvider_Send_SuccessWithInvalidJSON_FallsBackToGenerated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.MessageID)
	assert.Equal(t, "accepted", resp.Status)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestEmailProvider_Send_SuccessWithEmptyMessageID_FallsBackToGenerated(t *testing.T) {
	emptyIDResponse := provider.ProviderResponse{
		MessageID: "",
		Status:    "ok",
		Timestamp: "2026-03-08T10:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(emptyIDResponse)
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.MessageID)
	assert.Equal(t, "accepted", resp.Status)
}

func TestEmailProvider_Send_ConnectionError(t *testing.T) {
	p := newTestEmailProvider("http://127.0.0.1:1")
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, provider.ErrProviderConnectionFailed))
}

func TestEmailProvider_Send_ProviderRejected_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, provider.ErrProviderRejected))
}

func TestEmailProvider_Send_ProviderRejected_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, provider.ErrProviderRejected))
}

func TestEmailProvider_Send_EmptyResponseBody_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.MessageID)
	assert.Equal(t, "accepted", resp.Status)
	assert.NotEmpty(t, resp.Timestamp)
}

func TestEmailProvider_ImplementsNotificationProvider(t *testing.T) {
	var _ provider.NotificationProvider = (*EmailProvider)(nil)
}

func TestEmailProvider_Send_VerifiesRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-auth-key", r.Header.Get("x-ins-auth-key"))
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(provider.ProviderResponse{
			MessageID: "msg-headers",
			Status:    "accepted",
			Timestamp: "2026-03-08T10:00:00Z",
		})
	}))
	defer server.Close()

	p := newTestEmailProvider(server.URL)
	req := &provider.ProviderRequest{
		To:      "user@example.com",
		Channel: "email",
		Content: "Hello World",
	}

	resp, err := p.Send(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}
