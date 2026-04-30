package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mikhailjbs/chat-service/internal/domain/user"
)

type EventPayload struct {
	Event string            `json:"event"`
	Data  user.UserSnapshot `json:"data"`
}

func StartRedisSubscriber(ctx context.Context, redisURL string, db *gorm.DB) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	client := redis.NewClient(opts)

	pubsub := client.Subscribe(ctx, "events.users")

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()

		for msg := range ch {
			var payload EventPayload
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				log.Printf("Failed to unmarshal user event: %v", err)
				continue
			}

			if payload.Event == "UserCreated" || payload.Event == "UserUpdated" {
				snap := payload.Data
				// Upsert into user_snapshots
				if err := db.Clauses(clause.OnConflict{
					UpdateAll: true,
				}).Create(&snap).Error; err != nil {
					log.Printf("Failed to upsert user snapshot: %v", err)
				} else {
					log.Printf("Processed %s for user %s", payload.Event, snap.ID)
				}
			}
		}
	}()
}
