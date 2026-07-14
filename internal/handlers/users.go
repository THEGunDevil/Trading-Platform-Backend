package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetUserHandler fetches user by email
func GetUsersHandler(c *gin.Context) {
	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	params := gen.ListUsersPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	// 1️⃣ Fetch paginated users
	users, err := db.Q.ListUsersPaginated(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2️⃣ Count total users
	totalCount, err := db.Q.CountUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// 3️⃣ Build response
	var response []models.UserResponse
	for _, user := range users {
		response = append(response, models.UserResponse{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Email:          user.Email,
			PhoneNumber:    user.PhoneNumber,
			Role:           user.Role.String,
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,

			// 👈 optional: add field to struct
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(response),
		"total_count": totalCount,
		"total_pages": totalPages,
		"users":       response,
	})
}

func GetUserByIDHandler(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "userID not found in context"})
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid userID type"})
		return
	}

	// Optional: check if the middleware set banned_user flag
	// bannedUserFlag, _ := c.Get("banned_user")

	// 2️⃣ Fetch user from DB
	user, err := db.Q.GetUserByID(c.Request.Context(), pgtype.UUID{Bytes: userUUID, Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// 4️⃣ Build response
	resp := models.UserResponse{
		ID:             user.ID.Bytes,
		UserName:       user.UserName,
		Email:          user.Email,
		PhoneNumber:    user.PhoneNumber,
		Role:           user.Role.String,
		IsBanned:       user.IsBanned.Bool,
		IsPermanentBan: user.IsPermanentBan.Bool,
		BanReason:      user.BanReason.String,
		BanUntil:       &user.BanUntil.Time,
		CreatedAt:      user.CreatedAt.Time,
	}

	log.Printf("👤 Returning user data for user %v (banned: %v)", user.ID, user.IsBanned.Bool)
	c.JSON(http.StatusOK, resp)
}
func SearchUsersPaginatedHandler(c *gin.Context) {
	// Pagination
	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	// Search query (partial email)
	query := strings.TrimSpace(c.Query("email"))
	if query == "" {
		query = "" // or "*" depending on your query
	}

	log.Printf("🔍 Searching users: email='%s', page=%d", query, page)

	// SQLC params
	params := gen.SearchUsersByEmailWithPaginationParams{
		Column1: pgtype.Text{String: query, Valid: true}, // search term
		Limit:   int32(limit),
		Offset:  int32(offset),
	}

	// Execute search query
	rows, err := db.Q.SearchUsersByEmailWithPagination(c.Request.Context(), params)
	if err != nil {
		log.Printf("❌ Search error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	// Count total matching users
	totalCount, err := db.Q.CountUsersByEmail(c.Request.Context(), pgtype.Text{String: query, Valid: true})
	if err != nil {
		log.Printf("❌ Count error: %v", err)
		totalCount = 0
	}

	// Map response
	var result []models.UserResponse
	for _, user := range rows {
		result = append(result, models.UserResponse{
			ID:        user.ID.Bytes,
			UserName:  user.UserName,
			Email:     user.Email,
			Role:      user.Role.String,
			CreatedAt: user.CreatedAt.Time,
			// Add other fields if needed
		})
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(result),
		"total_count": totalCount,
		"total_pages": totalPages,
		"users":       result,
	})
}

// UpdateUserByIDHandler updates user by ID
func UpdateUserByIDHandler(c *gin.Context) {
	// Parse UUID
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	// Parse the incoming request
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params := gen.UpdateUserProfileParams{
		ID: pgtype.UUID{Bytes: parsedID, Valid: true},
	}

	if req.UserName != nil {
		params.UserName = pgtype.Text{String: *req.UserName, Valid: true}
	}

	if req.PhoneNumber != nil {
		params.PhoneNumber = pgtype.Text{String: *req.PhoneNumber, Valid: true}
	}
	// Save changes
	updatedUser, err := db.Q.UpdateUserProfile(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	// Response
	resp := models.UserResponse{
		ID:             updatedUser.ID.Bytes,
		UserName:       updatedUser.UserName,
		Email:          updatedUser.Email,
		PhoneNumber:    updatedUser.PhoneNumber,
		Role:           updatedUser.Role.String,
		IsBanned:       updatedUser.IsBanned.Bool,
		BanUntil:       &updatedUser.BanUntil.Time,
		BanReason:      updatedUser.BanReason.String,
		IsPermanentBan: updatedUser.IsPermanentBan.Bool,
		CreatedAt:      updatedUser.CreatedAt.Time,
	}

	c.JSON(http.StatusOK, resp)
}
func BanUserByIDHandler(c *gin.Context) {
	// Parse UUID
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Bind request
	var req models.BanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle BanUntil
	var banUntil pgtype.Timestamp
	if req.IsPermanentBan {
		// Permanent ban has no expiration
		banUntil = pgtype.Timestamp{Valid: false}
	} else if req.BanUntil != nil {
		banUntil = pgtype.Timestamp{Time: *req.BanUntil, Valid: true}
	} else {
		banUntil = pgtype.Timestamp{Valid: false}
	}

	// Update user ban
	params := gen.BanUserParams{
		ID:             pgtype.UUID{Bytes: parsedID, Valid: true},
		BanReason:      pgtype.Text{String: req.BanReason, Valid: true},
		BanUntil:       banUntil,
		IsPermanentBan: pgtype.Bool{Bool: req.IsPermanentBan, Valid: true},
	}

	err = db.Q.BanUser(c.Request.Context(), params)
	if err != nil {
		log.Printf("BanUser error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user ban status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "User banned successfully",
	})
}
