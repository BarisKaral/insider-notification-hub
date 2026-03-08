package notification

import (
	"net/http"

	"github.com/baris/notification-hub/pkg/errs"
)

var (
	ErrNotFound = errs.NewAppError(
		"NOTIFICATION_NOT_FOUND",
		"notification not found",
		http.StatusNotFound,
	)
	ErrCreateFailed = errs.NewAppError(
		"NOTIFICATION_CREATE_FAILED",
		"failed to create notification",
		http.StatusInternalServerError,
	)
	ErrCancelFailed = errs.NewAppError(
		"NOTIFICATION_CANCEL_FAILED",
		"notification cannot be cancelled in current state",
		http.StatusConflict,
	)
	ErrAlreadySent = errs.NewAppError(
		"NOTIFICATION_ALREADY_SENT",
		"notification has already been sent",
		http.StatusConflict,
	)
	ErrAlreadyCancelled = errs.NewAppError(
		"NOTIFICATION_ALREADY_CANCELLED",
		"notification has already been cancelled",
		http.StatusConflict,
	)
	ErrBatchTooLarge = errs.NewAppError(
		"BATCH_TOO_LARGE",
		"batch size exceeds maximum of 1000",
		http.StatusBadRequest,
	)
	ErrInvalidStatus = errs.NewAppError(
		"INVALID_STATUS",
		"invalid notification status",
		http.StatusBadRequest,
	)
	ErrDuplicateIdempotencyKey = errs.NewAppError(
		"DUPLICATE_IDEMPOTENCY_KEY",
		"a notification with this idempotency key already exists",
		http.StatusConflict,
	)
)
