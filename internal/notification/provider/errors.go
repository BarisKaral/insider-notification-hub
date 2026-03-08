package provider

import (
	"net/http"

	"github.com/baris/notification-hub/pkg/errs"
)

var (
	ErrProviderConnectionFailed = errs.NewAppError(
		"PROVIDER_CONNECTION_FAILED",
		"failed to connect to notification provider",
		http.StatusBadGateway,
	)
	ErrProviderRejected = errs.NewAppError(
		"PROVIDER_REJECTED",
		"notification provider rejected the request",
		http.StatusBadGateway,
	)
	ErrProviderTimeout = errs.NewAppError(
		"PROVIDER_TIMEOUT",
		"notification provider request timed out",
		http.StatusGatewayTimeout,
	)
)
