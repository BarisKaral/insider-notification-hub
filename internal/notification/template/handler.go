package template

import (
	"net/http"
	"strconv"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type handler struct {
	svc Service
}

var _ Handler = (*handler)(nil)

func NewHandler(svc Service) Handler {
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

func (h *handler) Create(c *fiber.Ctx) error {
	var req CreateRequest
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

	return response.Success(c, http.StatusCreated, ToResponse(t))
}

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

	return response.Success(c, http.StatusOK, ToResponse(t))
}

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
		"items":  ToResponseList(templates),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
	}

	var req UpdateRequest
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

	return response.Success(c, http.StatusOK, ToResponse(t))
}

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
