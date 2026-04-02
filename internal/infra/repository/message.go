package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mikhailjbs/chat-service/internal/domain/message"
)

// messageRepository persists chat messages in MongoDB.
type messageRepository struct {
	collection *mongo.Collection
}

func NewMessageRepository(client *mongo.Client, database string) message.Repository {
	if client == nil {
		panic("NewMessageRepository: mongo client is nil")
	}
	if database == "" {
		database = "chat_db"
	}
	coll := client.Database(database).Collection("messages")
	repo := &messageRepository{collection: coll}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *messageRepository) ensureIndexes(ctx context.Context) {
	// compound index for quick conversation lookups (latest first)
	idx1 := mongo.IndexModel{
		Keys: bson.D{
			{Key: "sender_id", Value: 1},
			{Key: "receiver_id", Value: 1},
			{Key: "createdAt", Value: -1},
		},
		Options: options.Index().SetBackground(true),
	}
	_, _ = r.collection.Indexes().CreateOne(ctx, idx1)

	// extended index with tie-breaker on _id to support stable keyset pagination
	idx2 := mongo.IndexModel{
		Keys: bson.D{
			{Key: "sender_id", Value: 1},
			{Key: "receiver_id", Value: 1},
			{Key: "createdAt", Value: -1},
			{Key: "_id", Value: -1},
		},
		Options: options.Index().SetBackground(true),
	}
	_, _ = r.collection.Indexes().CreateOne(ctx, idx2)
}

func (r *messageRepository) Create(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if msg == nil {
		return nil, errors.New("message payload is nil")
	}
	result, err := r.collection.InsertOne(ctx, msg)
	if err != nil {
		return nil, err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		msg.ID = oid
	}
	return msg, nil
}

func (r *messageRepository) MarkRead(ctx context.Context, ids []primitive.ObjectID, readerID string) error {
	if len(ids) == 0 {
		return nil
	}
	filter := bson.M{
		"_id":         bson.M{"$in": ids},
		"receiver_id": readerID,
	}
	update := bson.M{
		"$set": bson.M{
			"read":      true,
			"updatedAt": time.Now().UTC(),
		},
	}
	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

func (r *messageRepository) ListConversation(ctx context.Context, userID, peerID string, limit int64) ([]*message.Message, error) {
	filter := bson.M{
		"$or": bson.A{
			bson.M{"sender_id": userID, "receiver_id": peerID},
			bson.M{"sender_id": peerID, "receiver_id": userID},
		},
	}
	// Sort by createdAt desc, then _id desc for stable ordering
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var messages []*message.Message
	for cur.Next(ctx) {
		var msg message.Message
		if err := cur.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, cur.Err()
}

// ListConversationBefore returns older messages in a conversation before the given cursor.
// The result is ordered by createdAt desc, _id desc and limited by 'limit'.
func (r *messageRepository) ListConversationBefore(
	ctx context.Context,
	userID, peerID string,
	beforeCreatedAt time.Time,
	beforeID primitive.ObjectID,
	limit int64,
) ([]*message.Message, error) {
	// Conversation filter
	base := bson.M{
		"$or": bson.A{
			bson.M{"sender_id": userID, "receiver_id": peerID},
			bson.M{"sender_id": peerID, "receiver_id": userID},
		},
	}

	// Keyset condition: (createdAt < beforeCreatedAt) OR (createdAt == beforeCreatedAt AND _id < beforeID)
	// Using $or with an equality branch ensures deterministic page boundaries.
	keyset := bson.M{
		"$or": bson.A{
			bson.M{"createdAt": bson.M{"$lt": beforeCreatedAt}},
			bson.M{"createdAt": beforeCreatedAt, "_id": bson.M{"$lt": beforeID}},
		},
	}

	// Merge filters
	filter := bson.M{"$and": bson.A{base, keyset}}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}

	cur, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var messages []*message.Message
	for cur.Next(ctx) {
		var msg message.Message
		if err := cur.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, cur.Err()
}

func (r *messageRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*message.Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cur, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var messages []*message.Message
	for cur.Next(ctx) {
		var msg message.Message
		if err := cur.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, cur.Err()
}
