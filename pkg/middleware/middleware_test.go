package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupMiddleware_DoesNotPanic(t *testing.T) {
	app := fiber.New()
	assert.NotPanics(t, func() {
		SetupMiddleware(app)
	})
}

func TestSetupMiddleware_RequestIDHeaderPresent(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))
}

func TestSetupMiddleware_CORSHeadersPresent(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestSetupMiddleware_CORSAllowMethods(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	assert.Contains(t, allowMethods, "GET")
	assert.Contains(t, allowMethods, "POST")
	assert.Contains(t, allowMethods, "PUT")
	assert.Contains(t, allowMethods, "PATCH")
	assert.Contains(t, allowMethods, "DELETE")
	assert.Contains(t, allowMethods, "OPTIONS")
}

func TestSetupMiddleware_CORSAllowHeaders(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "X-Idempotency-Key")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowHeaders, "X-Idempotency-Key")
}

func TestSetupMiddleware_RecoverFromPanic(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Recover middleware should catch the panic and return 500
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSetupMiddleware_NormalRequestAfterPanic(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})
	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	// Trigger the panic
	req1 := httptest.NewRequest(http.MethodGet, "/panic", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	_ = resp1.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode)

	// Normal request should still work
	req2 := httptest.NewRequest(http.MethodGet, "/ok", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

func TestSetupMiddleware_RequestIDUniquePerRequest(t *testing.T) {
	app := fiber.New()
	SetupMiddleware(app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	id1 := resp1.Header.Get("X-Request-Id")
	_ = resp1.Body.Close()

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	id2 := resp2.Header.Get("X-Request-Id")
	_ = resp2.Body.Close()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}
