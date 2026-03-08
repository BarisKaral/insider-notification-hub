package httpclient

import "errors"

var (
	ErrRequestFailed  = errors.New("http request failed")
	ErrMaxRetries     = errors.New("max retries exceeded")
	ErrMarshalBody    = errors.New("failed to marshal request body")
)
