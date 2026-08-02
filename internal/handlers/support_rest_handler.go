package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/internal/ws"
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
	UserName, err := h.Queries.GetUserNameByID(c.Request.Context(), service.UUIDToPGType(userID))
	if err == nil && UserName != "" {
		userName = UserName
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

const (
	maxUploadSize = 8 << 20 // 8 MB
	uploadDir     = "./public/uploads"
)

// UploadResponse – ক্লায়েন্টকে যা ফেরত দেবো
type UploadResponse struct {
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// POST /support/upload
// func (h *SupportRestHandler) UploadFile(c *gin.Context) {
// 	// 1. অথেনটিকেশন চেক (অন্য হ্যান্ডলারগুলোর মতো)
// 	userID, ok := service.UserIDFromContext(c)
// 	if !ok {
// 		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
// 		return
// 	}
// 	_ = userID // পরে ব্যবহার করতে পারো, যেমন মালিকানা ট্র্যাক করতে

// 	// 2. ফাইলের সাইজ লিমিট (Gin‑এ MaxMultipartMemory সেট না করলে ডিফল্ট 32MB)
// 	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

// 	// 3. মাল্টিপার্ট ফর্ম থেকে ফাইল নেওয়া
// 	file, header, err := c.Request.FormFile("file")
// 	if err != nil {
// 		service.AbortWithError(c, http.StatusBadRequest, "No file provided")
// 		return
// 	}
// 	defer file.Close()

// 	// 4. ফাইল টাইপ ভ্যালিডেশন
// 	contentType := header.Header.Get("Content-Type")
// 	if !strings.HasPrefix(contentType, "image/") {
// 		service.AbortWithError(c, http.StatusBadRequest, "Only image files are allowed")
// 		return
// 	}

// 	// 5. এক্সটেনশন ও অনুমোদিত ফরম্যাট চেক
// 	ext := strings.ToLower(filepath.Ext(header.Filename))
// 	if ext == "" {
// 		ext = ".jpg"
// 	}
// 	allowedExts := map[string]bool{
// 		".jpg": true, ".jpeg": true, ".png": true,
// 		".gif": true, ".webp": true,
// 	}
// 	if !allowedExts[ext] {
// 		service.AbortWithError(c, http.StatusBadRequest, "Unsupported file type")
// 		return
// 	}

// 	// 6. ইউনিক ফাইলনেম তৈরি ও ডিরেক্টরি নিশ্চিত করা
// 	newName := uuid.New().String() + ext
// 	if err := os.MkdirAll(uploadDir, 0755); err != nil {
// 		log.Println("Failed to create upload dir:", err)
// 		service.AbortWithError(c, http.StatusInternalServerError, "Internal server error")
// 		return
// 	}

// 	destPath := filepath.Join(uploadDir, newName)
// 	dst, err := os.Create(destPath)
// 	if err != nil {
// 		log.Println("Failed to create destination file:", err)
// 		service.AbortWithError(c, http.StatusInternalServerError, "Internal server error")
// 		return
// 	}
// 	defer dst.Close()

// 	// 7. ফাইল কপি
// 	if _, err := io.Copy(dst, file); err != nil {
// 		log.Println("Failed to save file:", err)
// 		service.AbortWithError(c, http.StatusInternalServerError, "Internal server error")
// 		return
// 	}

// 	// 8. সফল রেসপন্স (Gin JSON)
// 	url := "/uploads/" + newName
// 	c.JSON(http.StatusOK, UploadResponse{
// 		Success: true,
// 		URL:     url,
// 	})
// }

func (h *SupportRestHandler) UploadToCloudinary(c *gin.Context) {
	// 1. অথেনটিকেশন
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = userID

	// 2. ফাইল সাইজ লিমিট
	const maxUploadSize = 8 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	// 3. ফাইল নেওয়া
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// 4. টাইপ ও এক্সটেনশন ভ্যালিডেশন
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		service.AbortWithError(c, http.StatusBadRequest, "Only image files are allowed")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".webp": true,
	}
	if !allowedExts[ext] {
		service.AbortWithError(c, http.StatusBadRequest, "Unsupported file type")
		return
	}

	// 5. Cloudinary ক্লায়েন্ট তৈরি (এনভায়রনমেন্ট ভেরিয়েবল থেকে)
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		service.AbortWithError(c, http.StatusInternalServerError, "Cloudinary configuration missing")
		return
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "Failed to initialize Cloudinary")
		return
	}

	// 6. ইউনিক পাব্লিক আইডি বানানো (ফোল্ডার সহ)
	publicID := "chat_uploads/" + uuid.New().String()

	// 7. আপলোড করা (io.Reader দিলেই হবে)
	ctx := context.Background()
	uniqueFilename := false
	overwrite := false
	uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		UniqueFilename: &uniqueFilename,
		Overwrite:      &overwrite,
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "Upload to Cloudinary failed")
		return
	}

	// 8. সিকিউর URL রিটার্ন
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"url":     uploadResult.SecureURL,
	})
}
