package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/internal/db"
	gen "github.com/internal/db/gen"
	"github.com/internal/models"
	"github.com/internal/service"
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
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,
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
	targetID, err := service.ParseUUIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// 1️⃣ Fetch user from DB
	user, err := db.Q.GetUserByID(c.Request.Context(), pgtype.UUID{Bytes: targetID, Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// 2️⃣ Fetch balances for this user
	balances, err := db.Q.ListBalances(c.Request.Context(), service.UUIDToPGType(targetID))
	if err != nil {
		balances = []gen.Balance{} // empty slice on error
	}

	// 3️⃣ Convert to models.Balance (JSON-friendly)
	balanceModels, err := service.ToBalanceModels(balances)
	if err != nil {
		log.Printf("⚠️ Failed to convert balances: %v", err)
		balanceModels = []models.Balance{} // fallback
	}
	userRes := models.UserResponse{
		ID:             user.ID.Bytes,
		UserName:       user.UserName,
		Email:          user.Email,
		Role:           user.Role.String,
		IsBanned:       user.IsBanned.Bool,
		BanUntil:       &user.BanUntil.Time,
		BanReason:      user.BanReason.String,
		IsPermanentBan: user.IsPermanentBan.Bool,
		CreatedAt:      user.CreatedAt.Time,
	}
	// 4️⃣ Build response
	resp := gin.H{
		"user":     userRes,
		"balances": balanceModels,
	}

	log.Printf("👤 Returning user data for user %v (banned: %v) with %d balances", user.ID, user.IsBanned.Bool, len(balanceModels))
	c.JSON(http.StatusOK, resp)
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
		IsBanned:       updatedUser.IsBanned.Bool,
		BanUntil:       &updatedUser.BanUntil.Time,
		BanReason:      updatedUser.BanReason.String,
		IsPermanentBan: updatedUser.IsPermanentBan.Bool,
		CreatedAt:      updatedUser.CreatedAt.Time,
	}

	c.JSON(http.StatusOK, resp)
}
