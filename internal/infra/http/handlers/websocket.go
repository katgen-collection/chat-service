package handlers

import (
	"context"

	"github.com/gofiber/websocket/v2"

	"mikhailjbs/chat-service/internal/infra/middleware"
	"mikhailjbs/chat-service/internal/infra/security"
	ws "mikhailjbs/chat-service/internal/infra/websocket"
)

// WebsocketHandler orchestrates websocket lifecycle within Fiber.
type WebsocketHandler interface {
	Handle(conn *websocket.Conn)
}

type websocketHandler struct {
	hub *ws.Hub
}

func NewWebsocketHandler(hub *ws.Hub) WebsocketHandler {
	return &websocketHandler{hub: hub}
}

// Handle godoc
// @Summary Connect to real-time chat
// @Description Establishes a WebSocket connection for real-time messaging.
// @Tags Realtime
// @Security BearerAuth
// @Param token query string false "JWT token if not provided in cookie/header"
// @Router /api/v1/ws [get]
func (h *websocketHandler) Handle(conn *websocket.Conn) {
	defer conn.Close()

	claims := resolveClaims(conn, middleware.DefaultClaimsContextKey)
	if claims == nil || claims.UserID == "" {
		_ = conn.WriteJSON(map[string]interface{}{
			"event": ws.EventError,
			"data":  map[string]string{"message": "unauthorized"},
		})
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h.hub.ServeConnection(ctx, claims.UserID, conn)
}

func resolveClaims(conn *websocket.Conn, key string) *security.ClaimsPayload {
	val := conn.Locals(key)
	if val == nil {
		return nil
	}
	if claims, ok := val.(*security.ClaimsPayload); ok {
		return claims
	}
	if claims, ok := val.(security.ClaimsPayload); ok {
		return &claims
	}
	return nil
}
