package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mikhailjbs/chat-service/internal/infra/http/handlers"
	"mikhailjbs/chat-service/internal/infra/httpclient"
)

// setupTestDB sets up an in-memory or throwaway DB for testing
// Since SQLite might not be available out of the box with GORM driver easily if not downloaded,
// we just mock the http client test the function that directly depends on interconnectivity.

func TestAcceptContactRequest_Interconnectivity(t *testing.T) {
	// 1. Mock the user-auth-service HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/sender-123", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		res := handlers.SuccessResponse{
			Status: 200,
			Data: httpclient.User{
				ID:       "sender-123",
				Username: "sender_user",
				Email:    "sender@test.com",
			},
		}
		json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/v1/users/receiver-456", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		res := handlers.SuccessResponse{
			Status: 200,
			Data: httpclient.User{
				ID:       "receiver-456",
				Username: "receiver_user",
				Email:    "receiver@test.com",
			},
		}
		json.NewEncoder(w).Encode(res)
	})

	mockAuthServer := httptest.NewServer(mux)
	defer mockAuthServer.Close()

	// 2. Setup userAuthClient
	authClient, err := httpclient.New(mockAuthServer.URL, mockAuthServer.Client())
	require.NoError(t, err)

	// We only test fetchUser and buildContact logic that was extracted or part of the repo
	// Since AcceptContactRequest operates on GORM db directly, let's explicitly test the interconnectivity parts
	
	ctx := context.Background()

	// Test fetchUser directly using the interconnected client
	repo := &userRepository{userClient: authClient} // Mocked DB is nil, but fetchUser doesn't use it

	sender, err := repo.fetchUser(ctx, "sender-123", "mytoken")
	require.NoError(t, err)
	assert.Equal(t, "sender-123", sender.ID)
	assert.Equal(t, "sender_user", sender.Username)

	receiver, err := repo.fetchUser(ctx, "receiver-456", "mytoken")
	require.NoError(t, err)
	assert.Equal(t, "receiver-456", receiver.ID)
	assert.Equal(t, "receiver_user", receiver.Username)

	// Test buildContact ensuring data maps tightly
	receiverContact, err := buildContact("receiver-456", sender)
	require.NoError(t, err)
	assert.Equal(t, "receiver-456", receiverContact.UserID)
	assert.Equal(t, "sender-123", receiverContact.ContactID)
	assert.Equal(t, "sender_user", receiverContact.Name) // falls back to username
}
