package template

import (
	"net/http"
	"time"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

type CreateRequest struct {
	Name    string `json:"name" validate:"required"`
	Channel string `json:"channel" validate:"required,oneof=sms email push"`
	Content string `json:"content" validate:"required"`
}

func (r *CreateRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return errs.NewAppError("VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
	}
	return nil
}

type UpdateRequest struct {
	Name    *string `json:"name,omitempty"`
	Channel *string `json:"channel,omitempty" validate:"omitempty,oneof=sms email push"`
	Content *string `json:"content,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return errs.NewAppError("VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
	}
	if r.Name == nil && r.Channel == nil && r.Content == nil {
		return errs.NewAppError("VALIDATION_ERROR", "at least one field must be provided", http.StatusBadRequest)
	}
	return nil
}

type TemplateResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Channel   string    `json:"channel"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ToResponse(t *Template) TemplateResponse {
	return TemplateResponse{
		ID:        t.ID,
		Name:      t.Name,
		Channel:   t.Channel,
		Content:   t.Content,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func ToResponseList(templates []*Template) []TemplateResponse {
	responses := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = ToResponse(t)
	}
	return responses
}
