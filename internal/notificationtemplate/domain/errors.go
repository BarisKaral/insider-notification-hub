package domain

import (
	"net/http"

	"github.com/bariskaral/insider-notification-hub/pkg/errs"
)

var (
	ErrNotificationTemplateNotFound = errs.NewAppError(
		"TEMPLATE_NOT_FOUND",
		"template not found",
		http.StatusNotFound,
	)
	ErrNotificationTemplateNameExists = errs.NewAppError(
		"TEMPLATE_NAME_EXISTS",
		"a template with this name already exists",
		http.StatusConflict,
	)
	ErrNotificationTemplateCreateFailed = errs.NewAppError(
		"TEMPLATE_CREATE_FAILED",
		"failed to create template",
		http.StatusInternalServerError,
	)
)
