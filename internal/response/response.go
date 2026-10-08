package response

import "github.com/gofiber/fiber/v2"

type Meta struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type APIResponse struct {
	StatusCode int         `json:"status_code"`
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Meta       *Meta       `json:"meta,omitempty"`
	Errors     interface{} `json:"errors,omitempty"`
}

func Success(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		StatusCode: statusCode,
		Success:    true,
		Message:    message,
		Data:       data,
	})
}

func SuccessWithMeta(c *fiber.Ctx, statusCode int, message string, data interface{}, meta *Meta) error {
	return c.Status(statusCode).JSON(APIResponse{
		StatusCode: statusCode,
		Success:    true,
		Message:    message,
		Data:       data,
		Meta:       meta,
	})
}

func Error(c *fiber.Ctx, statusCode int, message string, errors interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		StatusCode: statusCode,
		Success:    false,
		Message:    message,
		Errors:     errors,
	})
}
