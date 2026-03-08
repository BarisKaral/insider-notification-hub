package health

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// HealthHandler defines the HTTP handler interface for health check endpoints.
type HealthHandler interface {
	Health(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type healthHandler struct {
	service HealthService
}

var _ HealthHandler = (*healthHandler)(nil)

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(service HealthService) HealthHandler {
	return &healthHandler{
		service: service,
	}
}

// RegisterRoutes registers health check routes under the provided router.
func (h *healthHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/health", h.Health)
}

// Health handles GET /health.
func (h *healthHandler) Health(c *fiber.Ctx) error {
	resp := h.service.Check(c.Context())
	statusCode := http.StatusOK
	if resp.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	return c.Status(statusCode).JSON(resp)
}
