package response

import (
	"github.com/bariskaral/insider-notification-hub/pkg/errs"
	"github.com/gofiber/fiber/v2"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success(c *fiber.Ctx, statusCode int, data interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

func Error(c *fiber.Ctx, statusCode int, code, message string) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

func AppError(c *fiber.Ctx, err *errs.AppError) error {
	return Error(c, err.GetStatusCode(), err.Code, err.Message)
}
