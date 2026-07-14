package models

import (
	"mime/multipart"
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
	Bio                string     `json:"bio"` // added
	Role               string     `json:"role"`
	ProfileImgPublicID *string    `json:"profile_img_public_id"`
	ProfileImg         *string    `json:"profile_img"`
	TokenVersion       int        `json:"token_version"` // added
	IsBanned           bool       `json:"is_banned"`
	BanReason          string     `json:"ban_reason"`
	BanUntil           *time.Time `json:"ban_until"`        // optional, RFC3339 format
	IsPermanentBan     bool       `json:"is_permanent_ban"` // true = permanent ban

}

type UpdateUserRequest struct {
	UserName          *string               `form:"user_name"`
	Bio                *string               `form:"bio"` // added
	PhoneNumber        *string               `form:"phone_number"`
	ProfileImg         *multipart.FileHeader `form:"profile_img"`
	ProfileImgPublicID *string               `form:"profile_img_public_id"`
}

type UserResponse struct {
	ID                 uuid.UUID  `json:"id"`
	UserName          string     `json:"first_name"`
	Bio                string     `json:"bio"` // added
	Email              string     `json:"email"`
	PhoneNumber        string     `json:"phone_number"`
	CreatedAt          time.Time  `json:"created_at"`
	Role               string     `json:"role"`
	TokenVersion       int        `json:"token_version"` // added
	IsBanned           bool       `json:"is_banned"`
	BanReason          string     `json:"ban_reason"`
	BanUntil           *time.Time `json:"ban_until"` // nil = no ban date
	ProfileImgPublicID *string    `json:"profile_img_public_id"`
	ProfileImg         *string    `json:"profile_img"`
	IsPermanentBan     bool       `json:"is_permanent_ban"`
	LastActivity       time.Time  `json:"last_activity"`
	BooksReserved      int        `json:"books_reserved"`
	TotalReviews       int        `json:"total_reviews"`
	OverdueBooks       int        `json:"overdue_books"`
	ReadingStreak      int        `json:"reading_streak"`
	AllBorrowsCount    int        `json:"all_borrows_count"`    // true = permanent ban
	ActiveBorrowsCount int        `json:"active_borrows_count"` // true = permanent ban
}
type BanRequest struct {
	IsBanned       bool       `json:"is_banned"`
	BanReason      string     `json:"ban_reason"`
	BanUntil       *time.Time `json:"ban_until"`        // nullable
	IsPermanentBan bool       `json:"is_permanent_ban"` // true = permanent ban
}
