package handlers

import (
	"net/http"
	"time"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SupportRestHandler struct {
	Queries *gen.Queries
	Hub     *ws.SupportHub
}

// NewSupportRestHandler creates a new SupportRestHandler.
func NewSupportRestHandler(queries *gen.Queries, hub *ws.SupportHub) *SupportRestHandler {
	return &SupportRestHandler{Queries: queries, Hub: hub}
}

// POST /support/sessions
// POST /support/sessions
func (h *SupportRestHandler) CreateSession(c *gin.Context) {
    userID, ok := service.UserIDFromContext(c)
    if !ok {
        service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
        return
    }

    var req struct {
        Subject string `json:"subject" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        service.AbortWithError(c, http.StatusBadRequest, "subject is required")
        return
    }

    // ✅ Check if the user already has an open session
    existing, err := h.Queries.GetOpenSessionForUser(c.Request.Context(), service.UUIDToPGType(userID))
    if err == nil && existing.Status == "open" {
        // Already have an open session – just return it, no new notification
        c.JSON(http.StatusOK, existing)
        return
    }

    // No existing open session → create a new one
    session, err := h.Queries.CreateSupportSession(c.Request.Context(), gen.CreateSupportSessionParams{
        UserID:  service.UUIDToPGType(userID),
        Subject: req.Subject,
    })
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to create session")
        return
    }

    // Extract user name from context or fetch from DB
    var userName string
    user, err := h.Queries.GetUserByID(c.Request.Context(), service.UUIDToPGType(userID))
    if err == nil && user.UserName != "" {
        userName = user.UserName
    }

    // Notify agents only for truly new sessions
    h.Hub.TriggerNewSessionNotification(session, userID.String(), userName, req.Subject)

    c.JSON(http.StatusCreated, session)
}

// GET /support/sessions/open
func (h *SupportRestHandler) GetOpenSession(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	session, err := h.Queries.GetOpenSessionForUser(c.Request.Context(), service.UUIDToPGType(userID))
	if err != nil {
		// No open session – return null, not an error
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, session)
}

// GET /support/sessions/:id/messages
func (h *SupportRestHandler) GetMessages(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid session id")
		return
	}

	messages, err := h.Queries.ListSessionMessages(c.Request.Context(), service.UUIDToPGType(sessionID))
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch messages")
		return
	}

	c.JSON(http.StatusOK, messages)
}

// PATCH /support/messages/:id
// PATCH /support/messages/:id
func (h *SupportRestHandler) UpdateMessage(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid message id")
		return
	}

	var req struct {
		Content  string  `json:"content"`
		ImageURL *string `json:"image_url,omitempty"` // optional
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify ownership
	msg, err := h.Queries.GetSupportMessageByID(c.Request.Context(), service.UUIDToPGType(messageID))
	if err != nil {
		service.AbortWithError(c, http.StatusNotFound, "message not found")
		return
	}
	if service.PGTypeToUUID(msg.SenderID) != userID {
		service.AbortWithError(c, http.StatusForbidden, "you can only edit your own messages")
		return
	}

	// Determine new image: if request provided a new one, use it; otherwise keep existing
	newImage := msg.ImageUrl
	if req.ImageURL != nil {
		newImage = service.StringToPGTextNullable(*req.ImageURL)
	}

	updated, err := h.Queries.UpdateMessage(c.Request.Context(), gen.UpdateMessageParams{
		ID:       service.UUIDToPGType(messageID),
		Content:  service.StringToPGText(req.Content),
		ImageUrl: newImage,
		SenderID: service.UUIDToPGType(userID),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to update message")
		return
	}

	// Broadcast edit event
	var imageURLString string
	if req.ImageURL != nil {
		imageURLString = *req.ImageURL
	}
	h.Hub.BroadcastToSession(msg.SessionID.String(), &ws.WebSocketMessage{
		Type:      "message_updated",
		ID:        messageID.String(),
		SessionID: msg.SessionID.String(),
		Content:   req.Content,
		ImageURL:  imageURLString,
		SenderID:  userID.String(),
		IsAgent:   msg.IsAgent,
		Timestamp: time.Now(),
	})

	c.JSON(http.StatusOK, updated)
}

// DELETE /support/messages/:id
func (h *SupportRestHandler) DeleteMessage(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid message id")
		return
	}

	// Fetch message to verify ownership and get session ID
	msg, err := h.Queries.GetSupportMessageByID(c.Request.Context(), service.UUIDToPGType(messageID))
	if err != nil {
		service.AbortWithError(c, http.StatusNotFound, "message not found")
		return
	}
	if service.PGTypeToUUID(msg.SenderID) != userID {
		service.AbortWithError(c, http.StatusForbidden, "you can only delete your own messages")
		return
	}

	err = h.Queries.DeleteMessage(c.Request.Context(), gen.DeleteMessageParams{
		ID:       service.UUIDToPGType(messageID),
		SenderID: service.UUIDToPGType(userID),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to delete message")
		return
	}

	// Notify other participants
	h.Hub.BroadcastToSession(msg.SessionID.String(), &ws.WebSocketMessage{
		Type:      "message_deleted",
		ID:        messageID.String(),
		SessionID: msg.SessionID.String(),
		Timestamp: time.Now(),
	})

	c.Status(http.StatusNoContent)
}

// POST /support/upload
// func (h *SupportRestHandler) UploadImage(c *gin.Context) {
// 	// TODO: upload to S3/Cloudinary and return image URL
// 	c.JSON(http.StatusOK, gin.H{"url": "https://example.com/placeholder.jpg"})
// }
