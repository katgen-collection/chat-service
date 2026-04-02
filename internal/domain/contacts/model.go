package contacts

import "time"

type Contact struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	UserID       string    `gorm:"not null" json:"user_id"`
	ContactID    string    `gorm:"not null" json:"contact_id"`
	Name         string    `gorm:"type:varchar(100);not null" json:"name"`
	Username     string    `gorm:"type:varchar(50);not null" json:"username"`
	AssignedName string    `gorm:"type:varchar(100);default:''" json:"assigned_name"`
	Email        string    `gorm:"type:varchar(100);not null" json:"email"`
	Muted        bool      `gorm:"default:false" json:"muted"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ContactRequest struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	SenderID     string    `json:"sender_id"`
	ReceiverID   string    `json:"receiver_id"`
	SenderName   string    `json:"sender_name" gorm:"type:varchar(100)"`
	ReceiverName string    `json:"receiver_name" gorm:"type:varchar(100)"`
	Message      string    `json:"message"`
	Status       string    `json:"status" gorm:"default:'pending'"` // e.g., "pending", "accepted", "rejected"
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
