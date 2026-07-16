package models

import (
	"time"

	"github.com/google/uuid"
)

type Balance struct {
	UserID    uuid.UUID `json:"user_id"`
	Asset     string    `json:"asset"`
	Available string    `json:"available"` // string, not float — avoids precision loss over JSON
	Locked    string    `json:"locked"`
	UpdatedAt time.Time `json:"updated_at"`
}