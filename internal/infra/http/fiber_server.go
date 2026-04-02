package http

import (
	"mikhailjbs/chat-service/internal/infra/http/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewServer(corsAllowedOrigins string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "Chat Service",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsAllowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))
	app.Use(logger.New())
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return handlers.SendSuccess(c, fiber.StatusOK, "Service is healthy", nil)
	})

	return app
}
