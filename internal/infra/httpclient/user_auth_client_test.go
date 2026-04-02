package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mikhailjbs/chat-service/internal/infra/http/handlers"
)

// roundTripFunc allows us to mock http.Client's RoundTripper
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func mockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
		Timeout:   2 * time.Second,
	}
}

func TestGetUserByID_Success(t *testing.T) {
	mockUser := User{
		ID:       "123",
		Username: "testuser",
		Email:    "test@example.com",
	}

	successResponse := handlers.SuccessResponse{
		Status:  200,
		Message: "Success",
		Data:    mockUser,
	}

	body, err := json.Marshal(successResponse)
	require.NoError(t, err)

	client, err := New("http://mock-auth-url", mockHTTPClient(func(req *http.Request) *http.Response {
		assert.Equal(t, "http://mock-auth-url/api/v1/users/123", req.URL.String())
		assert.Equal(t, "Bearer mytoken", req.Header.Get("Authorization"))

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	}))
	require.NoError(t, err)

	result, err := client.GetUserByID(context.Background(), "123", "mytoken")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "123", result.User.ID)
	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "test@example.com", result.User.Email)
}

func TestGetUserByID_Error(t *testing.T) {
	errorResponse := handlers.ErrorResponse{
		Status: 404,
		Error:  "User not found",
	}

	body, err := json.Marshal(errorResponse)
	require.NoError(t, err)

	client, err := New("http://mock-auth-url", mockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	}))
	require.NoError(t, err)

	result, err := client.GetUserByID(context.Background(), "999", "mytoken")
	require.Error(t, err)
	require.Nil(t, result)

	// Custom structured error
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, "User not found", apiErr.Payload.Error)
	assert.Equal(t, 404, apiErr.Payload.Status)
}
