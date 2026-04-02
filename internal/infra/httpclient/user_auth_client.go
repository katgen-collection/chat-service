package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client provides typed helpers to talk with the user-auth-service HTTP API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new user-auth-service client. When httpClient is nil a sane default
// with a short timeout is used to avoid leaking goroutines on slow upstream calls.
func New(baseURL string, httpClient *http.Client) (*Client, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, errors.New("user-auth-service baseURL is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL:    strings.TrimRight(trimmed, "/"),
		httpClient: httpClient,
	}, nil
}

// User represents the subset of fields the chat-service cares about when
// referencing identities owned by the user-auth-service.
type User struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	FullName  string   `json:"full_name,omitempty"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	Status    string   `json:"status,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// SuccessResponse represents a successful API response envelope.
type SuccessResponse struct {
	Ok      bool        `json:"ok"`
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error API response envelope.
type ErrorResponse struct {
	Ok     bool   `json:"ok"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// GetUserResult bundles the typed user with the standardized envelope coming
// from the user-auth-service. Response.Data is overwritten with the typed user
// pointer for convenience.
type GetUserResult struct {
	User     *User
	Response SuccessResponse
}

// GetUserByID fetches a user from the user-auth-service. When bearerToken is
// empty the Authorization header is omitted which lets the caller opt-in to
// public endpoints.
func (c *Client) GetUserByID(ctx context.Context, userID string, bearerToken string) (*GetUserResult, error) {
	if c == nil {
		return nil, errors.New("user-auth-service client is nil")
	}
	if userID == "" {
		return nil, errors.New("userID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/api/v1/users/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build user request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call user-auth-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, parseAPIError(resp)
	}

	var envelope SuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode success payload: %w", err)
	}
	if envelope.Status == 0 {
		envelope.Status = resp.StatusCode
	}

	user, err := coerceUser(envelope.Data)
	if err != nil {
		return nil, err
	}
	envelope.Data = user

	return &GetUserResult{
		User:     user,
		Response: envelope,
	}, nil
}

// APIError captures structured errors emitted by the user-auth-service.
type APIError struct {
	Payload ErrorResponse
}

func (e *APIError) Error() string {
	if e == nil {
		return "user-auth-service error"
	}
	msg := e.Payload.Error
	if msg == "" {
		msg = "unknown user-auth-service error"
	}
	return fmt.Sprintf("user-auth-service: %s (status %d)", msg, e.Payload.Status)
}

func parseAPIError(resp *http.Response) error {
	var payload ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("user-auth-service unexpected status %d", resp.StatusCode)
	}
	if payload.Status == 0 {
		payload.Status = resp.StatusCode
	}
	return &APIError{Payload: payload}
}

func coerceUser(raw interface{}) (*User, error) {
	if raw == nil {
		return nil, errors.New("user-auth-service response did not include user data")
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal user payload: %w", err)
	}
	var user User
	if err := json.Unmarshal(bytes, &user); err != nil {
		return nil, fmt.Errorf("unmarshal user payload: %w", err)
	}
	return &user, nil
}
