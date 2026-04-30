package security

// ClaimsPayload is a lightweight struct to hold user data passed by the API Gateway via HTTP headers.
type ClaimsPayload struct {
	UserID   string
	Email    string
	Username string
	Roles    []string
}
