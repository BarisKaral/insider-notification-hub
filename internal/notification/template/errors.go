package template

import (
	"net/http"

	"github.com/baris/notification-hub/pkg/errs"
)

var (
	ErrTemplateNotFound = errs.NewAppError(
		"TEMPLATE_NOT_FOUND",
		"template not found",
		http.StatusNotFound,
	)
	ErrTemplateNameExists = errs.NewAppError(
		"TEMPLATE_NAME_EXISTS",
		"a template with this name already exists",
		http.StatusConflict,
	)
	ErrTemplateCreateFailed = errs.NewAppError(
		"TEMPLATE_CREATE_FAILED",
		"failed to create template",
		http.StatusInternalServerError,
	)
)
