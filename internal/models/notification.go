package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Notification represents a single notification for the user
type Notification struct {
	ID                uuid.UUID       `json:"id"`                  // Event ID
	UserID            uuid.UUID       `json:"user_id"`             // The user who sees this notification
	UserName          string          `json:"user_name,omitempty"` // optional
	ObjectID          *uuid.UUID      `json:"object_id,omitempty"` // related object, optional
	ObjectTitle       string          `json:"object_title,omitempty"`
	Type              string          `json:"type"`                 // e.g., BOOK_AVAILABLE, REMINDER
	NotificationTitle string          `json:"notification_title"`   // short title for UI
	Message           string          `json:"message"`              // full message
	Metadata          json.RawMessage `json:"metadata,omitempty"`   // optional JSONB data
	IsRead            bool            `json:"is_read"`              // read/unread status
	CreatedAt         time.Time       `json:"created_at"`           // creation timestamp
}

// SendNotificationRequest is used to create/send a new notification/event
type SendNotificationRequest struct {
	UserID            uuid.UUID       `json:"user_id" binding:"required"`
	UserName          string          `json:"user_name,omitempty"`
	ObjectID          *uuid.UUID      `json:"object_id,omitempty"`
	ObjectTitle       string          `json:"object_title,omitempty"`
	Type              string          `json:"type" binding:"required"`               // e.g., BOOK_AVAILABLE, REMINDER
	NotificationTitle string          `json:"notification_title" binding:"required"` // short title
	Message           string          `json:"message" binding:"required"`            // full message
	Metadata          json.RawMessage `json:"metadata,omitempty"`                    // optional extra info
}
