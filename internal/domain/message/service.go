package message

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidPayload   = errors.New("invalid message payload")
	ErrInvalidRecipient = errors.New("receiver_id is required")
)

// Repository captures the storage layer operations needed by the service.
type Repository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	MarkRead(ctx context.Context, ids []primitive.ObjectID, readerID string) error
	ListConversation(ctx context.Context, userID, peerID string, limit int64) ([]*Message, error)
	FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*Message, error)
	// ListConversationBefore returns messages older than the given cursor.
	ListConversationBefore(ctx context.Context, userID, peerID string, beforeCreatedAt time.Time, beforeID primitive.ObjectID, limit int64) ([]*Message, error)
}

// Service exposes message-domain behaviors consumed by transports.
type Service interface {
	SendMessage(ctx context.Context, senderID string, req *SendMessageRequest) (*Message, error)
	MarkMessagesRead(ctx context.Context, readerID string, ids []string) error
	GetRecentConversation(ctx context.Context, userID, peerID string, limit int64) ([]*Message, error)
	GetMessagesByIDs(ctx context.Context, ids []string) ([]*Message, error)
	// GetConversationBefore returns older messages using keyset pagination.
	GetConversationBefore(ctx context.Context, userID, peerID string, beforeCreatedAt time.Time, beforeID string, limit int64) ([]*Message, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SendMessage(ctx context.Context, senderID string, req *SendMessageRequest) (*Message, error) {
	if req == nil || req.Text == "" && req.Image == "" {
		return nil, ErrInvalidPayload
	}
	if req.ReceiverID == "" {
		return nil, ErrInvalidRecipient
	}
	if req.ReceiverID == senderID {
		return nil, ErrInvalidRecipient
	}

	now := time.Now().UTC()
	msg := &Message{
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		Text:       req.Text,
		Image:      req.Image,
		Read:       false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.repo.Create(ctx, msg)
}

func (s *service) MarkMessagesRead(ctx context.Context, readerID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, raw := range ids {
		id, err := primitive.ObjectIDFromHex(raw)
		if err != nil {
			return err
		}
		objectIDs = append(objectIDs, id)
	}

	return s.repo.MarkRead(ctx, objectIDs, readerID)
}

func (s *service) GetRecentConversation(ctx context.Context, userID, peerID string, limit int64) ([]*Message, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListConversation(ctx, userID, peerID, limit)
}

func (s *service) GetMessagesByIDs(ctx context.Context, ids []string) ([]*Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, raw := range ids {
		id, err := primitive.ObjectIDFromHex(raw)
		if err != nil {
			return nil, err
		}
		objectIDs = append(objectIDs, id)
	}

	return s.repo.FindByIDs(ctx, objectIDs)
}

func (s *service) GetConversationBefore(ctx context.Context, userID, peerID string, beforeCreatedAt time.Time, beforeID string, limit int64) ([]*Message, error) {
	if limit <= 0 {
		limit = 50
	}
	var oid primitive.ObjectID
	if beforeID != "" {
		var err error
		oid, err = primitive.ObjectIDFromHex(beforeID)
		if err != nil {
			return nil, err
		}
	}
	return s.repo.ListConversationBefore(ctx, userID, peerID, beforeCreatedAt, oid, limit)
}
