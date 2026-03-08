package notification

import (
	"net/http"
	"strconv"
	"time"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/logger"
	"github.com/baris/notification-hub/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// NotificationHandler defines the HTTP handler interface for notification endpoints.
type NotificationHandler interface {
	Create(c *fiber.Ctx) error
	CreateBatch(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	GetByBatchID(c *fiber.Ctx) error
	List(c *fiber.Ctx) error
	Cancel(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type notificationHandler struct {
	service  NotificationService
	producer NotificationProducer
}

var _ NotificationHandler = (*notificationHandler)(nil)

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(service NotificationService, producer NotificationProducer) NotificationHandler {
	return &notificationHandler{
		service:  service,
		producer: producer,
	}
}

// RegisterRoutes registers notification routes under the provided router group.
func (h *notificationHandler) RegisterRoutes(router fiber.Router) {
	notifications := router.Group("/notifications")
	notifications.Post("/", h.Create)
	notifications.Post("/batch", h.CreateBatch)
	notifications.Get("/batch/:batchId", h.GetByBatchID)
	notifications.Get("/:id", h.GetByID)
	notifications.Get("/", h.List)
	notifications.Patch("/:id/cancel", h.Cancel)
}

// Create handles POST /notifications.
func (h *notificationHandler) Create(c *fiber.Ctx) error {
	// Read optional idempotency key header.
	var idempotencyKey *string
	if key := c.Get("X-Idempotency-Key"); key != "" {
		idempotencyKey = &key
	}

	// Parse request body.
	var req NotificationCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Validate.
	if err := req.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	// Create notification via service.
	n, err := h.service.Create(c.Context(), req, idempotencyKey)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create notification")
	}

	// If not scheduled, publish to queue.
	if n.Status != NotificationStatusScheduled {
		if err := h.producer.Publish(c.Context(), n); err != nil {
			logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to publish notification")
		} else {
			if err := h.service.MarkAsQueued(c.Context(), n.ID); err != nil {
				logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to mark notification as queued")
			} else {
				n.Status = NotificationStatusQueued
			}
		}
	}

	return response.Success(c, http.StatusCreated, ToNotificationResponse(n))
}

// CreateBatch handles POST /notifications/batch.
func (h *notificationHandler) CreateBatch(c *fiber.Ctx) error {
	// Parse request body.
	var req NotificationBatchCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
	}

	// Validate.
	if err := req.Validate(); err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	// Create batch via service.
	notifications, batchID, err := h.service.CreateBatch(c.Context(), req)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create batch")
	}

	// Filter non-scheduled notifications and publish.
	var toPublish []*Notification
	for _, n := range notifications {
		if n.Status != NotificationStatusScheduled {
			toPublish = append(toPublish, n)
		}
	}

	if len(toPublish) > 0 {
		if err := h.producer.PublishBatch(c.Context(), toPublish); err != nil {
			logger.Error().Err(err).Str("batchID", batchID.String()).Msg("failed to publish batch notifications")
		} else {
			for _, n := range toPublish {
				if err := h.service.MarkAsQueued(c.Context(), n.ID); err != nil {
					logger.Error().Err(err).Str("notificationID", n.ID.String()).Msg("failed to mark notification as queued")
				} else {
					n.Status = NotificationStatusQueued
				}
			}
		}
	}

	return response.Success(c, http.StatusCreated, NotificationBatchResponse{
		BatchID:       batchID,
		Notifications: ToNotificationResponseList(notifications),
	})
}

// GetByID handles GET /notifications/:id.
func (h *notificationHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
	}

	n, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification")
	}

	return response.Success(c, http.StatusOK, ToNotificationResponse(n))
}

// GetByBatchID handles GET /notifications/batch/:batchId.
func (h *notificationHandler) GetByBatchID(c *fiber.Ctx) error {
	batchID, err := uuid.Parse(c.Params("batchId"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid batch ID")
	}

	notifications, err := h.service.GetByBatchID(c.Context(), batchID)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get batch notifications")
	}

	return response.Success(c, http.StatusOK, ToNotificationResponseList(notifications))
}

// List handles GET /notifications.
func (h *notificationHandler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	filter := NotificationListFilter{
		Status:  c.Query("status"),
		Channel: c.Query("channel"),
		Limit:   limit,
		Offset:  offset,
	}

	// Parse optional date filters.
	if startDate := c.Query("startDate"); startDate != "" {
		t, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_PARAM", "invalid startDate format, expected RFC3339")
		}
		filter.StartDate = &t
	}

	if endDate := c.Query("endDate"); endDate != "" {
		t, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_PARAM", "invalid endDate format, expected RFC3339")
		}
		filter.EndDate = &t
	}

	filter.Normalize()

	notifications, total, err := h.service.List(c.Context(), filter)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notifications")
	}

	return response.Success(c, http.StatusOK, NotificationPaginatedResponse{
		Items:  ToNotificationResponseList(notifications),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// Cancel handles PATCH /notifications/:id/cancel.
func (h *notificationHandler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
	}

	n, err := h.service.Cancel(c.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errs.AppError); ok {
			return response.AppError(c, appErr)
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to cancel notification")
	}

	return response.Success(c, http.StatusOK, ToNotificationResponse(n))
}
