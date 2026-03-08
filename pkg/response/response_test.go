package response

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess_ReturnsCorrectStatusAndBody(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, http.StatusOK, map[string]string{"name": "test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.True(t, apiResp.Success)
	assert.Nil(t, apiResp.Error)

	data, ok := apiResp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test", data["name"])
}

func TestSuccess_WithNilData(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, http.StatusOK, nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.True(t, apiResp.Success)
	assert.Nil(t, apiResp.Data)
	assert.Nil(t, apiResp.Error)
}

func TestSuccess_WithCreatedStatus(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		return Success(c, http.StatusCreated, map[string]int{"id": 42})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.True(t, apiResp.Success)
	assert.NotNil(t, apiResp.Data)
}

func TestError_ReturnsCorrectStatusAndBody(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid input")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	assert.Nil(t, apiResp.Data)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
	assert.Equal(t, "invalid input", apiResp.Error.Message)
}

func TestError_WithNotFoundStatus(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Error(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "NOT_FOUND", apiResp.Error.Code)
	assert.Equal(t, "resource not found", apiResp.Error.Message)
}

func TestError_WithInternalServerError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)
}

func TestAppError_ReturnsCorrectResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		appErr := errs.NewAppError("NOT_FOUND", "item not found", http.StatusNotFound)
		return AppError(c, appErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	assert.Nil(t, apiResp.Data)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "NOT_FOUND", apiResp.Error.Code)
	assert.Equal(t, "item not found", apiResp.Error.Message)
}

func TestAppError_WithBadRequest(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		appErr := errs.NewAppError("VALIDATION_ERROR", "email is required", http.StatusBadRequest)
		return AppError(c, appErr)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "VALIDATION_ERROR", apiResp.Error.Code)
	assert.Equal(t, "email is required", apiResp.Error.Message)
}

func TestAppError_WithInternalServerError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		appErr := errs.NewAppError("INTERNAL_ERROR", "unexpected failure", http.StatusInternalServerError)
		return AppError(c, appErr)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.False(t, apiResp.Success)
	require.NotNil(t, apiResp.Error)
	assert.Equal(t, "INTERNAL_ERROR", apiResp.Error.Code)
	assert.Equal(t, "unexpected failure", apiResp.Error.Message)
}

func TestSuccess_ContentTypeIsJSON(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestError_ContentTypeIsJSON(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Error(c, http.StatusBadRequest, "ERR", "msg")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestSuccess_WithSliceData(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, http.StatusOK, []string{"a", "b", "c"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var apiResp APIResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err)

	assert.True(t, apiResp.Success)
	assert.NotNil(t, apiResp.Data)

	items, ok := apiResp.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
}
