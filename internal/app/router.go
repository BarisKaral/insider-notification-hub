package app

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/baris/notification-hub/pkg/errs"
	"github.com/baris/notification-hub/pkg/middleware"
	"github.com/baris/notification-hub/pkg/response"
)

// setupRouter configures the Fiber application with all routes and middleware.
func (a *App) setupRouter() {
	app := fiber.New(fiber.Config{
		ErrorHandler: globalErrorHandler,
	})

	// Middlewares
	middleware.SetupMiddleware(app)

	// Health
	a.container.HealthHandler.RegisterRoutes(app)

	// Metrics
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// WebSocket — upgrade middleware, then handlers
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/notifications/:id", websocket.New(a.container.WSHub.HandleNotificationWS))
	app.Get("/ws/notifications/batch/:batchId", websocket.New(a.container.WSHub.HandleBatchWS))

	// API v1
	v1 := app.Group("/api/v1")
	a.container.NotificationHandler.RegisterRoutes(v1)
	a.container.TemplateHandler.RegisterRoutes(v1)

	a.fiber = app
}

// globalErrorHandler handles all unhandled errors from Fiber handlers.
func globalErrorHandler(c *fiber.Ctx, err error) error {
	// Handle fiber errors
	if e, ok := err.(*fiber.Error); ok {
		return response.Error(c, e.Code, "HTTP_ERROR", e.Message)
	}

	// Handle app errors
	if appErr, ok := err.(*errs.AppError); ok {
		return response.AppError(c, appErr)
	}

	// Default
	return response.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}
