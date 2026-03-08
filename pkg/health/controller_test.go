package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockHealthService implements HealthService for controller tests.
type mockHealthService struct {
	mock.Mock
}

func (m *mockHealthService) Check(ctx context.Context) HealthResponse {
	args := m.Called(ctx)
	return args.Get(0).(HealthResponse)
}

func TestHealthController_Health_Healthy(t *testing.T) {
	mockSvc := new(mockHealthService)
	controller := NewHealthController(mockSvc)

	expectedResp := HealthResponse{
		Status: "healthy",
		Checks: map[string]string{
			"database": "up",
			"rabbitmq": "up",
		},
	}

	mockSvc.On("Check", mock.Anything).Return(expectedResp)

	app := fiber.New()
	controller.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "healthy", body.Status)
	assert.Equal(t, "up", body.Checks["database"])
	assert.Equal(t, "up", body.Checks["rabbitmq"])

	mockSvc.AssertExpectations(t)
}

func TestHealthController_Health_Unhealthy(t *testing.T) {
	mockSvc := new(mockHealthService)
	controller := NewHealthController(mockSvc)

	expectedResp := HealthResponse{
		Status: "unhealthy",
		Checks: map[string]string{
			"database": "down",
			"rabbitmq": "up",
		},
	}

	mockSvc.On("Check", mock.Anything).Return(expectedResp)

	app := fiber.New()
	controller.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var body HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "unhealthy", body.Status)
	assert.Equal(t, "down", body.Checks["database"])
	assert.Equal(t, "up", body.Checks["rabbitmq"])

	mockSvc.AssertExpectations(t)
}

func TestHealthController_RegisterRoutes(t *testing.T) {
	mockSvc := new(mockHealthService)
	controller := NewHealthController(mockSvc)

	app := fiber.New()
	controller.RegisterRoutes(app)

	routes := app.GetRoutes()
	found := false
	for _, route := range routes {
		if route.Path == "/health" && route.Method == http.MethodGet {
			found = true
			break
		}
	}

	assert.True(t, found, "expected GET /health route to be registered")
}

func TestNewHealthController_ReturnsNonNil(t *testing.T) {
	mockSvc := new(mockHealthService)
	controller := NewHealthController(mockSvc)
	assert.NotNil(t, controller)
}

func TestHealthController_Health_AllDown(t *testing.T) {
	mockSvc := new(mockHealthService)
	controller := NewHealthController(mockSvc)

	expectedResp := HealthResponse{
		Status: "unhealthy",
		Checks: map[string]string{
			"database": "down",
			"rabbitmq": "down",
		},
	}

	mockSvc.On("Check", mock.Anything).Return(expectedResp)

	app := fiber.New()
	controller.RegisterRoutes(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var body HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "unhealthy", body.Status)
	assert.Equal(t, "down", body.Checks["database"])
	assert.Equal(t, "down", body.Checks["rabbitmq"])

	mockSvc.AssertExpectations(t)
}
