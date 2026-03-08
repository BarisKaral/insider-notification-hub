package httpclient

import "errors"

var (
	ErrHTTPRequestFailed = errors.New("http request failed")
	ErrHTTPMaxRetries    = errors.New("max retries exceeded")
	ErrHTTPMarshalBody   = errors.New("failed to marshal request body")
)
