package message

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a chat message persisted in MongoDB.
// SenderID and ReceiverID are stored as strings to align with the UUID-based user IDs
// already used in the contacts domain, while MongoDB still owns the object identifier.
type Message struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Text       string             `bson:"text,omitempty" json:"text"`
	SenderID   string             `bson:"sender_id" json:"sender_id"`
	ReceiverID string             `bson:"receiver_id,omitempty" json:"receiver_id,omitempty"`
	Read       bool               `bson:"read" json:"read"`
	Image      string             `bson:"image,omitempty" json:"image,omitempty"`
	CreatedAt  time.Time          `bson:"createdAt" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updated_at"`
}
