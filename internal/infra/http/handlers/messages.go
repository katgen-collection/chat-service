package handlers

import (
    "encoding/base64"
    "encoding/json"
    "strconv"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "mikhailjbs/chat-service/internal/domain/message"
    "mikhailjbs/chat-service/internal/infra/middleware"
)

// MessageHandler exposes HTTP endpoints for message history.
type MessageHandler struct {
    messages message.Service
}

func NewMessageHandler(messages message.Service) MessageHandler {
    return MessageHandler{messages: messages}
}

// historyCursor is encoded as base64 JSON in the `cursor` query param.
// Example (JSON before base64): {"t":"2025-01-01T12:00:00Z","id":"65a1c3..."}
type historyCursor struct {
    T  string `json:"t"`  // RFC3339Nano timestamp
    ID string `json:"id"` // Mongo ObjectID hex
}

func encodeCursor(t time.Time, id primitive.ObjectID) string {
    if id == (primitive.ObjectID{}) {
        return ""
    }
    cur := historyCursor{T: t.UTC().Format(time.RFC3339Nano), ID: id.Hex()}
    raw, _ := json.Marshal(cur)
    return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(s string) (time.Time, string, error) {
    if s == "" {
        return time.Time{}, "", nil
    }
    raw, err := base64.RawURLEncoding.DecodeString(s)
    if err != nil {
        return time.Time{}, "", err
    }
    var cur historyCursor
    if err := json.Unmarshal(raw, &cur); err != nil {
        return time.Time{}, "", err
    }
    t, err := time.Parse(time.RFC3339Nano, cur.T)
    if err != nil {
        return time.Time{}, "", err
    }
    return t, cur.ID, nil
}

type historyResponse struct {
    Messages   []map[string]interface{} `json:"messages"`
    NextCursor string                   `json:"next_cursor,omitempty"`
    HasMore    bool                     `json:"has_more"`
}

// GetHistory returns a page of older messages for the conversation with `peer_id`.
// Query params:
// - peer_id: required peer user id
// - cursor: optional base64 cursor token (from previous response)
// - limit: optional integer, default 50
// GetHistory godoc
// @Summary Get message history
// @Description Fetches a paginated page of older messages for a conversation.
// @Tags Messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param peer_id query string true "Peer User ID"
// @Param cursor query string false "Base64 cursor token for pagination"
// @Param limit query int false "Number of messages to fetch (default 50, max 200)"
// @Success 200 {object} handlers.SuccessResponse{data=handlers.historyResponse}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/messages/history [get]
func (h MessageHandler) GetHistory(c *fiber.Ctx) error {
    claims, ok := middleware.ClaimsFromContext(c)
    if !ok || claims.UserID == "" {
        return SendError(c, fiber.StatusUnauthorized, "unauthorized")
    }

    peerID := c.Query("peer_id")
    if peerID == "" {
        return SendError(c, fiber.StatusBadRequest, "peer_id is required")
    }

    limit := int64(50)
    if v := c.Query("limit"); v != "" {
        if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 200 {
            limit = n
        }
    }

    cursor := c.Query("cursor")
    var msgs []*message.Message
    var err error
    if cursor == "" {
        // initial page: latest messages
        msgs, err = h.messages.GetRecentConversation(c.Context(), claims.UserID, peerID, limit)
        if err != nil {
            return SendError(c, fiber.StatusInternalServerError, err.Error())
        }
    } else {
        // keyset page: fetch older than cursor
        beforeT, beforeID, err := decodeCursor(cursor)
        if err != nil {
            return SendError(c, fiber.StatusBadRequest, "invalid cursor")
        }
        msgs, err = h.messages.GetConversationBefore(c.Context(), claims.UserID, peerID, beforeT, beforeID, limit)
        if err != nil {
            return SendError(c, fiber.StatusInternalServerError, err.Error())
        }
    }

    // Build response DTOs (messages already sorted desc by createdAt)
    out := make([]map[string]interface{}, 0, len(msgs))
    var next string
    if len(msgs) > 0 {
        // next cursor points to the oldest item in this page
        oldest := msgs[len(msgs)-1]
        next = encodeCursor(oldest.CreatedAt, oldest.ID)
    }
    for _, m := range msgs {
        id := ""
        if m.ID != (primitive.ObjectID{}) {
            id = m.ID.Hex()
        }
        out = append(out, map[string]interface{}{
            "id":          id,
            "sender_id":   m.SenderID,
            "receiver_id": m.ReceiverID,
            "text":        m.Text,
            "image":       m.Image,
            "read":        m.Read,
            "created_at":  m.CreatedAt.UTC().Format(time.RFC3339Nano),
            "updated_at":  m.UpdatedAt.UTC().Format(time.RFC3339Nano),
        })
    }

    resp := historyResponse{
        Messages:   out,
        NextCursor: next,
        HasMore:    int64(len(msgs)) == limit,
    }
    return SendSuccess(c, fiber.StatusOK, "history", resp)
}
