package message

// SendMessageRequest represents a payload for a new outbound message.
type SendMessageRequest struct {
	ReceiverID string `json:"receiver_id"`
	Text       string `json:"text"`
	Image      string `json:"image,omitempty"`
}

// ReadReceiptRequest captures message IDs a user has acknowledged.
type ReadReceiptRequest struct {
	MessageIDs []string `json:"message_ids"`
}
