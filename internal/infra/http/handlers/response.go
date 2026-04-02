package handlers

import "github.com/gofiber/fiber/v2"

type SuccessResponse struct {
	Ok      bool        `json:"ok"`
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Ok     bool   `json:"ok"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

func SendSuccess(c *fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(SuccessResponse{
		Ok:      true,
		Status:  status,
		Message: message,
		Data:    data,
	})
}

func SendError(c *fiber.Ctx, status int, errMessage string) error {
	return c.Status(status).JSON(ErrorResponse{
		Ok:     false,
		Status: status,
		Error:  errMessage,
	})
}
