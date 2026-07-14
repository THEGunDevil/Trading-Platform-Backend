package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                 uuid.UUID  `json:"id"`
	UserName          string     `json:"user_name"`
	Email              string     `json:"email"`
	PhoneNumber        string     `json:"phone_number"`
	Password           string     `json:"password"`
	ConfirmPassword    string     `json:"confirm_password"`
	Role               string     `json:"role"`
	TokenVersion       int        `json:"token_version"` // added
	IsBanned           bool       `json:"is_banned"`
	BanReason          string     `json:"ban_reason"`
	BanUntil           *time.Time `json:"ban_until"`        // optional, RFC3339 format
	IsPermanentBan     bool       `json:"is_permanent_ban"` // true = permanent ban

}

type UpdateUserRequest struct {
	UserName          *string               `form:"user_name"`
	PhoneNumber        *string               `form:"phone_number"`
}

type UserResponse struct {
	ID                 uuid.UUID  `json:"id"`
	UserName          string     `json:"first_name"`
	Email              string     `json:"email"`
	PhoneNumber        string     `json:"phone_number"`
	CreatedAt          time.Time  `json:"created_at"`
	Role               string     `json:"role"`
	TokenVersion       int        `json:"token_version"` // added
	IsBanned           bool       `json:"is_banned"`
	BanReason          string     `json:"ban_reason"`
	BanUntil           *time.Time `json:"ban_until"` // nil = no ban date
	IsPermanentBan     bool       `json:"is_permanent_ban"`
	LastActivity       time.Time  `json:"last_activity"`
}
type BanRequest struct {
	IsBanned       bool       `json:"is_banned"`
	BanReason      string     `json:"ban_reason"`
	BanUntil       *time.Time `json:"ban_until"`        // nullable
	IsPermanentBan bool       `json:"is_permanent_ban"` // true = permanent ban
}
