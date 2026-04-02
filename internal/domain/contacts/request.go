package contacts

type ContactQueryParams struct {
	UserID		string	 `json:"user_id"`
	Limit			int		   `json:"limit"`
	Offset		int		   `json:"offset"`
	Search    string   `json:"search,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	Order     string   `json:"order,omitempty"` // "asc" or "desc"
}

type ContactRequestQueryParams struct {
	Limit			 int		   `json:"limit"`
	Offset		 int		   `json:"offset"`
	Status     string    `json:"status,omitempty"` // e.g., "pending", "accepted", "rejected"
	SortBy		 string    `json:"sort_by,omitempty"`
	Order      string    `json:"order,omitempty"` // "asc" or "desc"
	SenderID	 string    `json:"sender_id,omitempty"`
	ReceiverID string    `json:"receiver_id,omitempty"`
}

type CreateContactRequest struct {
	SenderID    string   `json:"sender_id"`
	ReceiverID	string   `json:"receiver_id"`
	Message		  string   `json:"message"`
}

type UpdateContactRequest struct {
	Status    string   `json:"status"` // e.g., "accepted", "rejected"
}

type UpdateContact struct {
	AssignedName string `json:"assigned_name,omitempty"`
	Muted        *bool  `json:"muted,omitempty"`
}
