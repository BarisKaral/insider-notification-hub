package notification

import (
	"net/http"

	"github.com/baris/notification-hub/pkg/errs"
)

var (
	ErrNotificationNotFound = errs.NewAppError(
		"NOTIFICATION_NOT_FOUND",
		"notification not found",
		http.StatusNotFound,
	)
	ErrNotificationCreateFailed = errs.NewAppError(
		"NOTIFICATION_CREATE_FAILED",
		"failed to create notification",
		http.StatusInternalServerError,
	)
	ErrNotificationCancelFailed = errs.NewAppError(
		"NOTIFICATION_CANCEL_FAILED",
		"notification cannot be cancelled in current state",
		http.StatusConflict,
	)
	ErrNotificationAlreadySent = errs.NewAppError(
		"NOTIFICATION_ALREADY_SENT",
		"notification has already been sent",
		http.StatusConflict,
	)
	ErrNotificationAlreadyCancelled = errs.NewAppError(
		"NOTIFICATION_ALREADY_CANCELLED",
		"notification has already been cancelled",
		http.StatusConflict,
	)
	ErrNotificationBatchTooLarge = errs.NewAppError(
		"BATCH_TOO_LARGE",
		"batch size exceeds maximum of 1000",
		http.StatusBadRequest,
	)
	ErrNotificationInvalidStatus = errs.NewAppError(
		"INVALID_STATUS",
		"invalid notification status",
		http.StatusBadRequest,
	)
	ErrNotificationDuplicateIdempotencyKey = errs.NewAppError(
		"DUPLICATE_IDEMPOTENCY_KEY",
		"a notification with this idempotency key already exists",
		http.StatusConflict,
	)
)
