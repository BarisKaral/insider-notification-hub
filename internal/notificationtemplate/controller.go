package notificationtemplate

import (
	"net/http"
	"strconv"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NotificationTemplateController interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type controller struct {
	svc NotificationTemplateService
}

var _ NotificationTemplateController = (*controller)(nil)

func NewNotificationTemplateController(svc NotificationTemplateService) NotificationTemplateController {
	return &controller{svc: svc}
}

func (h *controller) RegisterRoutes(router fiber.Router) {
	templates := router.Group("/notification-templates")
	templates.Post("/", h.Create)
	templates.Get("/:id", h.GetByID)
	templates.Get("/", h.List)
	templates.Put("/:id", h.Update)
	templates.Delete("/:id", h.Delete)
}

// Create handles POST /notification-templates.
// @Summary Create a notification template
// @Description Create a new notification template.
// @Tags NotificationTemplates
// @Accept json
// @Produce json
// @Param request body NotificationTemplateCreateRequest true "Template payload"
// @Success 201 {object} response.APIResponse{data=NotificationTemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notification-templates [post]
func (h *controller) Create(c *fiber.Ctx) error {
	var req NotificationTemplateCreateRequest
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

	return response.Success(c, http.StatusCreated, ToNotificationTemplateResponse(t))
}

// GetByID handles GET /notification-templates/:id.
// @Summary Get notification template by ID
// @Description Retrieve a single notification template by its UUID.
// @Tags NotificationTemplates
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 200 {object} response.APIResponse{data=NotificationTemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notification-templates/{id} [get]
func (h *controller) GetByID(c *fiber.Ctx) error {
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

	return response.Success(c, http.StatusOK, ToNotificationTemplateResponse(t))
}

// List handles GET /notification-templates.
// @Summary List notification templates
// @Description List all notification templates with pagination.
// @Tags NotificationTemplates
// @Produce json
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notification-templates [get]
func (h *controller) List(c *fiber.Ctx) error {
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
		"items":  ToNotificationTemplateResponseList(templates),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Update handles PUT /notification-templates/:id.
// @Summary Update a notification template
// @Description Update an existing notification template by its UUID. At least one field must be provided.
// @Tags NotificationTemplates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Param request body NotificationTemplateUpdateRequest true "Template update payload"
// @Success 200 {object} response.APIResponse{data=NotificationTemplateResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notification-templates/{id} [put]
func (h *controller) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
	}

	var req NotificationTemplateUpdateRequest
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

	return response.Success(c, http.StatusOK, ToNotificationTemplateResponse(t))
}

// Delete handles DELETE /notification-templates/:id.
// @Summary Delete a notification template
// @Description Delete a notification template by its UUID.
// @Tags NotificationTemplates
// @Param id path string true "Template ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notification-templates/{id} [delete]
func (h *controller) Delete(c *fiber.Ctx) error {
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
