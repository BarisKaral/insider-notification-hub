package app

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/bariskaral/insider-notification-hub/docs"
	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/bariskaral/insider-notification-hub/pkg/middleware"
	"github.com/bariskaral/insider-notification-hub/pkg/response"
)

// setupRouter configures the Fiber application with all routes and middleware.
func (a *App) setupRouter() {
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: globalErrorHandler,
	})

	// OpenTelemetry middleware — traces ALL HTTP requests automatically.
	fiberApp.Use(otelfiber.Middleware())

	// Middlewares
	middleware.SetupMiddleware(fiberApp)

	// Health
	a.container.HealthController.RegisterRoutes(fiberApp)

	// Metrics
	fiberApp.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Swagger
	fiberApp.Get("/swagger/*", fiberSwagger.WrapHandler)

	// WebSocket — upgrade middleware, then handlers
	fiberApp.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	fiberApp.Get("/ws/notifications/:id", websocket.New(a.container.WSHub.HandleNotificationWS))
	fiberApp.Get("/ws/notifications/batch/:batchId", websocket.New(a.container.WSHub.HandleBatchWS))

	// API v1
	v1 := fiberApp.Group("/api/v1")
	a.container.NotificationController.RegisterRoutes(v1)
	a.container.NotificationTemplateController.RegisterRoutes(v1)

	a.fiberApp = fiberApp
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
