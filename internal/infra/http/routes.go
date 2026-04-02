package http

import (
	"mikhailjbs/chat-service/internal/infra/http/handlers"
	"mikhailjbs/chat-service/internal/infra/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/gofiber/websocket/v2"
)

func RegisterRoutes(
	app *fiber.App,
	contactHandler handlers.ContactHandler,
	contactRequestHandler handlers.ContactRequestHandler,
	messageHandler handlers.MessageHandler,
	wsHandler handlers.WebsocketHandler,
	authzMiddleware *middleware.AuthMiddleware,
) {
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api")
	v1 := api.Group("/v1")

	contacts := v1.Group("/contacts", authzMiddleware.Require(middleware.Policy{
		Roles: []string{"user", "admin"},
	}))
	contacts.Get("/", contactHandler.GetContacts)
	contacts.Get("/:id", contactHandler.GetContact)
	contacts.Put("/:id", contactHandler.UpdateContact)
	contacts.Delete("/:id", contactHandler.DeleteContact)

	contactRequest := v1.Group("/contact-requests", authzMiddleware.Require(middleware.Policy{
		Roles: []string{"user", "admin"},
	}))
	contactRequest.Post("/", contactRequestHandler.CreateContactRequest)
	contactRequest.Get("/:id", contactRequestHandler.GetContactRequest)
	contactRequest.Get("/", contactRequestHandler.GetContactRequests)
	contactRequest.Put("/:id", contactRequestHandler.UpdateContactRequest)
	contactRequest.Delete("/:id", contactRequestHandler.DeleteContactRequest)

	ws := v1.Group("/ws", authzMiddleware.Require(middleware.Policy{
		Roles: []string{"user", "admin"},
	}))
	ws.Use(func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	ws.Get("/", websocket.New(wsHandler.Handle))

	messages := v1.Group("/messages", authzMiddleware.Require(middleware.Policy{
		Roles: []string{"user", "admin"},
	}))
	messages.Get("/history", messageHandler.GetHistory)
}
