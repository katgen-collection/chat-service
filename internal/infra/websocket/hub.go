package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"mikhailjbs/chat-service/internal/domain/contacts"
	"mikhailjbs/chat-service/internal/domain/message"
)

const (
	// Outbound events
	EventGetOnlineUsers     = "getOnlineUsers"
	EventMutualStatusUpdate = "mutualStatusUpdate"
	EventMessageIncoming        = "message:incoming"
	EventMessageAck             = "message:ack"
	EventMessageRead            = "message:read"
	EventContactRequestIncoming = "contact_request:incoming"
	EventContactRequestUpdated  = "contact_request:updated"
	EventError                  = "error"

	// Inbound events
	EventSendMessage = "message:send"
	EventMarkRead    = "message:mark_read"
)

// inboundEnvelope is the message format clients send to the server.
// {"event":"message:send","data":{...}}
// Keeping this compact makes it easy for web and mobile clients to integrate.
type inboundEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// outboundEnvelope is what the server broadcasts back to clients.
type outboundEnvelope struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data,omitempty"`
}

// Hub manages active websocket clients and orchestrates message fan-out.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*client // keyed by userID
	contacts contacts.Service
	messages message.Service
	log      *logrus.Logger
}

func NewHub(contactsSvc contacts.Service, messageSvc message.Service, log *logrus.Logger) *Hub {
	return &Hub{
		clients:  make(map[string]*client),
		contacts: contactsSvc,
		messages: messageSvc,
		log:      log,
	}
}

// ServeConnection wires a websocket connection to the Hub lifecycle.
func (h *Hub) ServeConnection(ctx context.Context, userID string, conn *websocket.Conn) {
	c := h.register(userID, conn)
	defer func() {
		h.unregister(userID)
		_ = h.notifyMutuals(userID, "offline")
	}()

	// Initial presence snapshot: only online mutual contacts.
	if online, err := h.onlineMutuals(userID); err == nil {
		_ = c.send(EventGetOnlineUsers, online)
	} else {
		h.log.WithError(err).Warn("websocket: unable to compute online mutuals")
	}

	// Signal that this user is online to their mutual contacts.
	_ = h.notifyMutuals(userID, "online")

	for {
		var incoming inboundEnvelope
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}

		switch incoming.Event {
		case EventSendMessage:
			h.handleSendMessage(ctx, c, incoming.Data)
		case EventMarkRead:
			h.handleMarkRead(ctx, c, incoming.Data)
		default:
			_ = c.send(EventError, map[string]string{"message": "unknown event"})
		}
	}
}

func (h *Hub) handleSendMessage(ctx context.Context, c *client, raw json.RawMessage) {
	var payload message.SendMessageRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		_ = c.send(EventError, map[string]string{"message": "invalid send payload"})
		return
	}

	// Basic validation: receiver must be provided and not self
	if payload.ReceiverID == "" || payload.ReceiverID == c.userID {
		_ = c.send(EventError, map[string]string{"message": "invalid receiver_id"})
		return
	}

	// Ensure receiver exists in caller's contacts to avoid sending to non-existent users
	contactsList, err := h.contacts.ListContacts(&contacts.ContactQueryParams{UserID: c.userID})
	if err != nil {
		_ = c.send(EventError, map[string]string{"message": "unable to validate receiver"})
		return
	}
	found := false
	for _, ct := range contactsList {
		if ct != nil && ct.ContactID == payload.ReceiverID {
			found = true
			break
		}
	}
	if !found {
		_ = c.send(EventError, map[string]string{"message": "receiver not found in contacts"})
		return
	}

	msg, err := h.messages.SendMessage(ctx, c.userID, &payload)
	if err != nil {
		_ = c.send(EventError, map[string]string{"message": err.Error()})
		return
	}

	dto := toDTO(msg)

	// Ack the sender quickly.
	_ = c.send(EventMessageAck, dto)

	// Fan-out to receiver if connected.
	if receiver := h.getClient(payload.ReceiverID); receiver != nil {
		_ = receiver.send(EventMessageIncoming, dto)
	}
}

func (h *Hub) handleMarkRead(ctx context.Context, c *client, raw json.RawMessage) {
	var payload message.ReadReceiptRequest
	if err := json.Unmarshal(raw, &payload); err != nil {
		_ = c.send(EventError, map[string]string{"message": "invalid read payload"})
		return
	}

	if err := h.messages.MarkMessagesRead(ctx, c.userID, payload.MessageIDs); err != nil {
		_ = c.send(EventError, map[string]string{"message": err.Error()})
		return
	}

	// Notify sender(s) whose messages were read.
	msgs, err := h.messages.GetMessagesByIDs(ctx, payload.MessageIDs)
	if err != nil {
		_ = c.send(EventError, map[string]string{"message": err.Error()})
		return
	}

	for _, m := range msgs {
		sender := h.getClient(m.SenderID)
		if sender != nil {
			_ = sender.send(EventMessageRead, map[string]interface{}{
				"message_id": m.ID.Hex(),
				"reader_id":  c.userID,
			})
		}
	}

	// Confirm back to the reader as well.
	_ = c.send(EventMessageRead, map[string]interface{}{
		"message_ids": payload.MessageIDs,
		"reader_id":   c.userID,
	})
}

// register adds the client to the hub and returns the wrapped client.
func (h *Hub) register(userID string, conn *websocket.Conn) *client {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := &client{userID: userID, conn: conn}
	h.clients[userID] = c
	return c
}

func (h *Hub) unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[userID]; ok {
		_ = c.conn.Close()
		delete(h.clients, userID)
	}
}

func (h *Hub) getClient(userID string) *client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[userID]
}

func (h *Hub) onlineMutuals(userID string) ([]string, error) {
	contactsList, err := h.contacts.ListContacts(&contacts.ContactQueryParams{UserID: userID})
	if err != nil {
		return nil, err
	}

	online := make([]string, 0, len(contactsList))
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range contactsList {
		if _, ok := h.clients[c.ContactID]; ok {
			online = append(online, c.ContactID)
		}
	}
	return online, nil
}

func (h *Hub) notifyMutuals(userID string, status string) error {
	mutuals, err := h.onlineMutuals(userID)
	if err != nil {
		return err
	}
	for _, id := range mutuals {
		if peer := h.getClient(id); peer != nil {
			_ = peer.send(EventMutualStatusUpdate, map[string]string{
				"userId": userID,
				"status": status,
			})
		}
	}
	return nil
}

// SendToUser sends an arbitrary event to a specific connected user.
func (h *Hub) SendToUser(userID string, event string, data interface{}) error {
	if c := h.getClient(userID); c != nil {
		return c.send(event, data)
	}
	return nil
}

// NotifyNewContact cross-notifies two newly connected mutuals of their online status.
func (h *Hub) NotifyNewContact(userA, userB string) {
	cA := h.getClient(userA)
	cB := h.getClient(userB)

	if cA != nil && cB != nil {
		_ = cA.send(EventMutualStatusUpdate, map[string]string{
			"userId": userB,
			"status": "online",
		})
		_ = cB.send(EventMutualStatusUpdate, map[string]string{
			"userId": userA,
			"status": "online",
		})
	}
}

// messageDTO is a trimmed message representation safe for websocket clients.
type messageDTO struct {
	ID         string `json:"id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Text       string `json:"text"`
	Image      string `json:"image,omitempty"`
	Read       bool   `json:"read"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func toDTO(m *message.Message) *messageDTO {
	id := ""
	if m.ID != (primitive.ObjectID{}) {
		id = m.ID.Hex()
	}
	return &messageDTO{
		ID:         id,
		SenderID:   m.SenderID,
		ReceiverID: m.ReceiverID,
		Text:       m.Text,
		Image:      m.Image,
		Read:       m.Read,
		CreatedAt:  m.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:  m.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// client wraps a websocket connection with a write lock.
type client struct {
	userID string
	conn   *websocket.Conn
	mu     sync.Mutex
}

func (c *client) send(event string, data interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(outboundEnvelope{Event: event, Data: data})
}
