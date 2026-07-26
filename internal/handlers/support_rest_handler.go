package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/ws"
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

	session, err := h.Queries.CreateSupportSession(c.Request.Context(), gen.CreateSupportSessionParams{
		UserID:  service.UUIDToPGType(userID),
		Subject: req.Subject,
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Extract user name from context (set by your auth middleware)
	userName, _ := c.Get("user_name")
	nameStr := "User"
	if userName != nil {
		nameStr = userName.(string)
	}

	h.Hub.TriggerNewSessionNotification(session, userID.String(), nameStr, req.Subject)

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

// POST /support/upload
// func (h *SupportRestHandler) UploadImage(c *gin.Context) {
// 	// TODO: upload to S3/Cloudinary and return image URL
// 	c.JSON(http.StatusOK, gin.H{"url": "https://example.com/placeholder.jpg"})
// }