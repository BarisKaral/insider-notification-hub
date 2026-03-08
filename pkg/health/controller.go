package health

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// HealthController defines the HTTP controller interface for health check endpoints.
type HealthController interface {
	Health(c *fiber.Ctx) error
	RegisterRoutes(router fiber.Router)
}

type healthController struct {
	service HealthService
}

var _ HealthController = (*healthController)(nil)

// NewHealthController creates a new HealthController.
func NewHealthController(service HealthService) HealthController {
	return &healthController{
		service: service,
	}
}

// RegisterRoutes registers health check routes under the provided router.
func (h *healthController) RegisterRoutes(router fiber.Router) {
	router.Get("/health", h.Health)
}

// Health handles GET /health.
func (h *healthController) Health(c *fiber.Ctx) error {
	resp := h.service.Check(c.Context())
	statusCode := http.StatusOK
	if resp.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	return c.Status(statusCode).JSON(resp)
}
