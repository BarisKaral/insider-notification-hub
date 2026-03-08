package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/service"
	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/response"
)

// NotificationController defines the HTTP controller interface for notification endpoints.
type NotificationController interface {
	Create(c *fiber.Ctx) error
	CreateBatch(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	GetByBatchID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Cancel(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type notificationController struct {
	notificationService           service.NotificationService
	notificationProcessingService service.NotificationProcessingService
}

var _ NotificationController = (*notificationController)(nil)

// NewNotificationController creates a new NotificationController.
func NewNotificationController(notificationService service.NotificationService, notificationProcessingService service.NotificationProcessingService) NotificationController {
	return &notificationController{
		notificationService:           notificationService,
		notificationProcessingService: notificationProcessingService,
	}
}

// RegisterRoutes registers notification routes under the provided router group.
func (h *notificationController) RegisterRoutes(router fiber.Router) {
	notifications := router.Group("/notifications")
	notifications.Post("/", h.Create)
	notifications.Post("/batch", h.CreateBatch)
	notifications.Get("/batch/:batchId", h.GetByBatchID)
	notifications.Get("/:id", h.GetByID)
	notifications.Get("/", h.List)
	notifications.Patch("/:id/cancel", h.Cancel)
}

// Create handles POST /notifications.
// @Summary Create a notification
// @Description Create a new notification. Either content or templateId with variables must be provided.
// @Tags Notifications
// @Accept json
// @Produce json
// @Param X-Idempotency-Key header string false "Idempotency key for deduplication"
// @Param request body domain.NotificationCreateRequest true "Notification payload"
// @Success 201 {object} response.APIResponse{data=domain.NotificationResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications [post]
func (h *notificationController) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Read optional idempotency key header.
	var idempotencyKey *string
	if key := c.Get("X-Idempotency-Key"); key != "" {
		idempotencyKey = &key
	}

	// Parse request body.
	var request domain.NotificationCreateRequest
	if err := c.BodyParser(&request); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Validate.
	if err := request.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	notification, err := h.notificationProcessingService.Create(ctx, request, idempotencyKey)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create notification")
	}

	return response.Success(c, http.StatusCreated, domain.ToNotificationResponse(notification))
}

// CreateBatch handles POST /notifications/batch.
// @Summary Create batch notifications
// @Description Create multiple notifications in a single request (max 1000).
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body domain.NotificationBatchCreateRequest true "Batch notification payload"
// @Success 201 {object} response.APIResponse{data=domain.NotificationBatchResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/batch [post]
func (h *notificationController) CreateBatch(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Parse request body.
	var request domain.NotificationBatchCreateRequest
	if err := c.BodyParser(&request); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Validate.
	if err := request.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	notifications, batchID, err := h.notificationProcessingService.CreateBatch(ctx, request)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create batch")
	}

	return response.Success(c, http.StatusCreated, domain.NotificationBatchResponse{
		BatchID:       batchID,
		Notifications: domain.ToNotificationResponseList(notifications),
	})
}

// GetByID handles GET /notifications/:id.
// @Summary Get notification by ID
// @Description Retrieve a single notification by its UUID.
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID (UUID)"
// @Success 200 {object} response.APIResponse{data=domain.NotificationResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id} [get]
func (h *notificationController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
	}

	notification, err := h.notificationService.GetByID(c.UserContext(), id)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification")
	}

	return response.Success(c, http.StatusOK, domain.ToNotificationResponse(notification))
}

// GetByBatchID handles GET /notifications/batch/:batchId.
// @Summary Get notifications by batch ID
// @Description Retrieve all notifications belonging to a specific batch.
// @Tags Notifications
// @Produce json
// @Param batchId path string true "Batch ID (UUID)"
// @Success 200 {object} response.APIResponse{data=[]domain.NotificationResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/batch/{batchId} [get]
func (h *notificationController) GetByBatchID(c *fiber.Ctx) error {
	batchID, err := uuid.Parse(c.Params("batchId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid batch ID")
	}

	notifications, err := h.notificationService.GetByBatchID(c.UserContext(), batchID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get batch notifications")
	}

	return response.Success(c, http.StatusOK, domain.ToNotificationResponseList(notifications))
}

// List handles GET /notifications.
// @Summary List notifications
// @Description List notifications with optional filters for status, channel, and date range.
// @Tags Notifications
// @Produce json
// @Param status query string false "Filter by status (pending, queued, sent, failed, cancelled)"
// @Param channel query string false "Filter by channel (sms, email, push)"
// @Param startDate query string false "Filter by start date (RFC3339)"
// @Param endDate query string false "Filter by end date (RFC3339)"
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} response.APIResponse{data=domain.NotificationPaginatedResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications [get]
func (h *notificationController) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	filter := domain.NotificationListFilter{
		Status:  c.Query("status"),
		Channel: c.Query("channel"),
		Limit:   limit,
		Offset:  offset,
	}

	// Parse optional date filters.
	if startDate := c.Query("startDate"); startDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_PARAM", "invalid startDate format, expected RFC3339")
		}
		filter.StartDate = &parsedTime
	}

	if endDate := c.Query("endDate"); endDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_PARAM", "invalid endDate format, expected RFC3339")
		}
		filter.EndDate = &parsedTime
	}

	filter.Normalize()

	notifications, total, err := h.notificationService.List(c.UserContext(), filter)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notifications")
	}

	return response.Success(c, http.StatusOK, domain.NotificationPaginatedResponse{
		Items:  domain.ToNotificationResponseList(notifications),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// Cancel handles PATCH /notifications/:id/cancel.
// @Summary Cancel a notification
// @Description Cancel a pending or scheduled notification by its UUID.
// @Tags Notifications
// @Produce json
// @Param id path string true "Notification ID (UUID)"
// @Success 200 {object} response.APIResponse{data=domain.NotificationResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id}/cancel [patch]
func (h *notificationController) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
	}

	notification, err := h.notificationService.Cancel(c.UserContext(), id)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to cancel notification")
	}

	return response.Success(c, http.StatusOK, domain.ToNotificationResponse(notification))
}
