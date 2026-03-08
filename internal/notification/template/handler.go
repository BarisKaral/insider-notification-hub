package template

import (
	"net/http"
	"strconv"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TemplateHandler interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type handler struct {
	svc TemplateService
}

var _ TemplateHandler = (*handler)(nil)

func NewTemplateHandler(svc TemplateService) TemplateHandler {
	return &handler{svc: svc}
}

func (h *handler) RegisterRoutes(router fiber.Router) {
	templates := router.Group("/templates")
	templates.Post("/", h.Create)
	templates.Get("/:id", h.GetByID)
	templates.Get("/", h.List)
	templates.Put("/:id", h.Update)
	templates.Delete("/:id", h.Delete)
}

// Create handles POST /templates.
// @Summary Create a template
// @Description Create a new notification template.
// @Tags Templates
// @Accept json
// @Produce json
// @Param request body TemplateCreateRequest true "Template payload"
// @Success 201 {object} response.APIResponse{data=TemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /templates [post]
func (h *handler) Create(c *fiber.Ctx) error {
	var req TemplateCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if err := req.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	t, err := h.svc.Create(c.Context(), req)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create template")
	}

	return response.Success(c, http.StatusCreated, ToTemplateResponse(t))
}

// GetByID handles GET /templates/:id.
// @Summary Get template by ID
// @Description Retrieve a single template by its UUID.
// @Tags Templates
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 200 {object} response.APIResponse{data=TemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /templates/{id} [get]
func (h *handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
	}

	t, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get template")
	}

	return response.Success(c, http.StatusOK, ToTemplateResponse(t))
}

// List handles GET /templates.
// @Summary List templates
// @Description List all templates with pagination.
// @Tags Templates
// @Produce json
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /templates [get]
func (h *handler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	templates, total, err := h.svc.List(c.Context(), limit, offset)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list templates")
	}

	return response.Success(c, http.StatusOK, fiber.Map{
		"items":  ToTemplateResponseList(templates),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Update handles PUT /templates/:id.
// @Summary Update a template
// @Description Update an existing template by its UUID. At least one field must be provided.
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Param request body TemplateUpdateRequest true "Template update payload"
// @Success 200 {object} response.APIResponse{data=TemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /templates/{id} [put]
func (h *handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
	}

	var req TemplateUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	if err := req.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	t, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update template")
	}

	return response.Success(c, http.StatusOK, ToTemplateResponse(t))
}

// Delete handles DELETE /templates/:id.
// @Summary Delete a template
// @Description Delete a template by its UUID.
// @Tags Templates
// @Param id path string true "Template ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /templates/{id} [delete]
func (h *handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete template")
	}

	return c.SendStatus(http.StatusNoContent)
}
