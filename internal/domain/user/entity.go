package user

import "time"

type UserSnapshot struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email"`
	Fullname  string    `json:"fullname"`
	Username  string    `json:"username"`
	Avatar    *string   `json:"avatar"`
	Role      string    `json:"role"`
	UpdatedAt time.Time `json:"updated_at"`
}
