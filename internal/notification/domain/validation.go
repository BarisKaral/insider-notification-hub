package domain

import (
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Channel content character limits.
const (
	maxSMSLength   = 160
	maxEmailLength = 10000
	maxPushLength  = 256
)

// Normalize applies defaults to NotificationCreateRequest fields.
func (r *NotificationCreateRequest) Normalize() {
	if r.Priority == "" {
		r.Priority = string(NotificationPriorityNormal)
	}
}

// Validate normalizes defaults and performs struct-level validation and custom business rules.
func (r *NotificationCreateRequest) Validate() error {
	r.Normalize()

	// Struct tag validation.
	if err := validate.Struct(r); err != nil {
		return errs.NewAppError("VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
	}

	// Content XOR TemplateID: exactly one must be provided.
	hasContent := r.Content != nil && *r.Content != ""
	hasTemplate := r.TemplateID != nil

	if hasContent && hasTemplate {
		return errs.NewAppError(
			"VALIDATION_ERROR",
			"content and templateId are mutually exclusive; provide one or the other",
			http.StatusBadRequest,
		)
	}
	if !hasContent && !hasTemplate {
		return errs.NewAppError(
			"VALIDATION_ERROR",
			"either content or templateId must be provided",
			http.StatusBadRequest,
		)
	}

	// Channel-specific content length limits using rune count for proper Unicode support.
	if hasContent {
		limit := channelContentLimit(NotificationChannel(r.Channel))
		if utf8.RuneCountInString(*r.Content) > limit {
			return errs.NewAppError(
				"VALIDATION_ERROR",
				fmt.Sprintf("content exceeds maximum length of %d for channel %s", limit, r.Channel),
				http.StatusBadRequest,
			)
		}
	}

	return nil
}

// channelContentLimit returns the maximum content length for a given channel.
func channelContentLimit(ch NotificationChannel) int {
	switch ch {
	case NotificationChannelSMS:
		return maxSMSLength
	case NotificationChannelEmail:
		return maxEmailLength
	case NotificationChannelPush:
		return maxPushLength
	default:
		return maxEmailLength
	}
}

// Validate validates the batch request and each individual notification.
func (r *NotificationBatchCreateRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return errs.NewAppError("VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
	}

	for i := range r.Notifications {
		if err := r.Notifications[i].Validate(); err != nil {
			msg := err.Error()
			if appErr, ok := err.(*errs.AppError); ok {
				msg = appErr.Message
			}
			return errs.NewAppError(
				"VALIDATION_ERROR",
				fmt.Sprintf("notification[%d]: %s", i, msg),
				http.StatusBadRequest,
			)
		}
	}

	return nil
}

// Normalize applies default values and caps to the filter.
func (f *NotificationListFilter) Normalize() {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
