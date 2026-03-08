package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:        5 * time.Second,
		MaxRetries:     0,
		RetryDelay:     10 * time.Millisecond,
		DefaultHeaders: nil,
	}
}

func TestNewHTTPClient_ReturnsNonNil(t *testing.T) {
	c := NewHTTPClient(newTestConfig())
	assert.NotNil(t, c)
}

func TestGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.Get(context.Background(), server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok"}`, string(body))
}

func TestPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		err := json.Unmarshal(body, &payload)
		require.NoError(t, err)
		assert.Equal(t, "test", payload["name"])

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.Post(context.Background(), server.URL, map[string]string{"name": "test"}, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestPost_WithBodyMarshalingComplex(t *testing.T) {
	type payload struct {
		Name  string   `json:"name"`
		Tags  []string `json:"tags"`
		Count int      `json:"count"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p payload
		err := json.Unmarshal(body, &p)
		require.NoError(t, err)
		assert.Equal(t, "item", p.Name)
		assert.Equal(t, []string{"a", "b"}, p.Tags)
		assert.Equal(t, 5, p.Count)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.Post(context.Background(), server.URL, payload{
		Name:  "item",
		Tags:  []string{"a", "b"},
		Count: 5,
	}, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDo_RetryOnServerError(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	resp, err := c.Get(context.Background(), server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestDo_MaxRetriesExceeded(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	resp, err := c.Get(context.Background(), server.URL, nil)

	// On last attempt with 500, the response IS returned (not retried further).
	// The code: if response.StatusCode >= 500 && attempt < c.maxRetries
	// On last attempt (attempt == maxRetries), condition is false, so it returns the 500 response.
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Total calls: initial + maxRetries = 3
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestDo_MaxRetriesExceeded_ConnectionError(t *testing.T) {
	// Use an address that will refuse connection to trigger actual errors
	config := HTTPClientConfig{
		Timeout:    100 * time.Millisecond,
		MaxRetries: 1,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	_, err := c.Get(context.Background(), "http://127.0.0.1:1", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHTTPMaxRetries))
}

func TestDo_ContextCancellationDuringRetry(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 5,
		RetryDelay: 500 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay, during the retry wait
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := c.Get(ctx, server.URL, nil)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	// Should have been called at most once (first attempt fails, then context cancelled during retry delay)
	assert.LessOrEqual(t, atomic.LoadInt32(&callCount), int32(2))
}

func TestDo_MarshalError(t *testing.T) {
	config := newTestConfig()
	c := NewHTTPClient(config)

	// Channels cannot be marshaled to JSON
	body := make(chan int)

	_, err := c.Do(context.Background(), http.MethodPost, "http://localhost:9999", body, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHTTPMarshalBody))
}

func TestDo_DefaultHeadersApplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "my-app", r.Header.Get("X-App-Name"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 0,
		RetryDelay: 10 * time.Millisecond,
		DefaultHeaders: map[string]string{
			"Authorization": "Bearer token123",
			"X-App-Name":    "my-app",
		},
	}
	c := NewHTTPClient(config)

	resp, err := c.Get(context.Background(), server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDo_CustomHeadersOverrideDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer overridden", r.Header.Get("Authorization"))
		assert.Equal(t, "my-app", r.Header.Get("X-App-Name"))
		assert.Equal(t, "extra-val", r.Header.Get("X-Extra"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 0,
		RetryDelay: 10 * time.Millisecond,
		DefaultHeaders: map[string]string{
			"Authorization": "Bearer default",
			"X-App-Name":    "my-app",
		},
	}
	c := NewHTTPClient(config)

	customHeaders := map[string]string{
		"Authorization": "Bearer overridden",
		"X-Extra":       "extra-val",
	}

	resp, err := c.Get(context.Background(), server.URL, customHeaders)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGet_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Empty(t, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.Get(context.Background(), server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDo_ContentTypeHeaderAlwaysSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.Do(context.Background(), http.MethodPut, server.URL, nil, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDo_InvalidMethod(t *testing.T) {
	c := NewHTTPClient(newTestConfig())

	// Invalid method with a space will cause NewRequestWithContext to fail
	_, err := c.Do(context.Background(), "BAD METHOD", "http://localhost", nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHTTPRequestFailed))
}

func TestDo_RetryWithPostBody(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		err := json.Unmarshal(body, &payload)
		require.NoError(t, err)
		assert.Equal(t, "value", payload["key"])

		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	resp, err := c.Post(context.Background(), server.URL, map[string]string{"key": "value"}, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestDo_NonRetryable4xxError(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	resp, err := c.Get(context.Background(), server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	// 4xx errors should not be retried
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestDo_NilHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 0,
		RetryDelay: 10 * time.Millisecond,
	}
	c := NewHTTPClient(config)

	resp, err := c.Do(context.Background(), http.MethodGet, server.URL, nil, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
