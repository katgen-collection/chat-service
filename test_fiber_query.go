package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

type QueryParams struct {
	SenderID string `json:"sender_id"`
}

func main() {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		var q QueryParams
		c.QueryParser(&q)
		return c.SendString(fmt.Sprintf("SenderID: '%s'", q.SenderID))
	})

	app.Listen(":3050")
}
