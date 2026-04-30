package contacts

type ContactQueryParams struct {
	UserID		string	 `json:"user_id" query:"user_id"`
	Limit			int		   `json:"limit" query:"limit"`
	Offset		int		   `json:"offset" query:"offset"`
	Search    string   `json:"search,omitempty" query:"search"`
	SortBy    string   `json:"sort_by,omitempty" query:"sort_by"`
	Order     string   `json:"order,omitempty" query:"order"` // "asc" or "desc"
}

type ContactRequestQueryParams struct {
	Limit			 int		   `json:"limit" query:"limit"`
	Offset		 int		   `json:"offset" query:"offset"`
	Status     string    `json:"status,omitempty" query:"status"` // e.g., "pending", "accepted", "rejected"
	SortBy		 string    `json:"sort_by,omitempty" query:"sort_by"`
	Order      string    `json:"order,omitempty" query:"order"` // "asc" or "desc"
	SenderID	 string    `json:"sender_id,omitempty" query:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty" query:"receiver_id"`
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
