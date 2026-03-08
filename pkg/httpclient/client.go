package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client interface {
	Do(ctx context.Context, method, url string, body interface{}, headers map[string]string) (*http.Response, error)
	Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error)
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error)
}

type client struct {
	httpClient     *http.Client
	maxRetries     int
	retryDelay     time.Duration
	defaultHeaders map[string]string
}

func NewClient(cfg Config) Client {
	return &client{
		httpClient:     &http.Client{Timeout: cfg.Timeout},
		maxRetries:     cfg.MaxRetries,
		retryDelay:     cfg.RetryDelay,
		defaultHeaders: cfg.DefaultHeaders,
	}
}

func (c *client) Do(ctx context.Context, method, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMarshalBody, err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay):
			}
			// Reset body reader for retry
			if body != nil {
				data, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(data)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
		}

		req.Header.Set("Content-Type", "application/json")
		for k, v := range c.defaultHeaders {
			req.Header.Set(k, v)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrRequestFailed, err)
			continue
		}

		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			resp.Body.Close()
			lastErr = fmt.Errorf("%w: status %d", ErrRequestFailed, resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("%w: %v", ErrMaxRetries, lastErr)
}

func (c *client) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, url, nil, headers)
}

func (c *client) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, url, body, headers)
}
