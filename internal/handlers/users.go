package handlers

// import (
// 	"log"
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
// 	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"github.com/jackc/pgx/v5/pgtype"
// )

// // GetUserHandler fetches user by email
// func GetUsersHandler(c *gin.Context) {
// 	page := 1
// 	limit := 10

// 	if p := c.Query("page"); p != "" {
// 		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
// 			page = parsed
// 		}
// 	}
// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
// 			limit = parsed
// 		}
// 	}

// 	offset := (page - 1) * limit

// 	params := gen.ListUsersPaginatedParams{
// 		Limit:  int32(limit),
// 		Offset: int32(offset),
// 	}

// 	// 1️⃣ Fetch paginated users
// 	users, err := db.Q.ListUsersPaginated(c.Request.Context(), params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// 2️⃣ Count total users
// 	totalCount, err := db.Q.CountUsers(c.Request.Context())
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

// 	// 3️⃣ Build response
// 	var response []models.UserResponse
// 	for _, user := range users {
// 		activeBorrowsCount, err := db.Q.CountActiveBorrowsByUserID(
// 			c.Request.Context(),
// 			pgtype.UUID{Bytes: user.ID.Bytes, Valid: true},
// 		)
// 		if err != nil {
// 			log.Printf("Failed to count active borrows for user %v: %v", user.ID, err)
// 			activeBorrowsCount = 0
// 		}

// 		allBorrowsCount, err := db.Q.CountBorrowedBooksByUserID(
// 			c.Request.Context(),
// 			pgtype.UUID{Bytes: user.ID.Bytes, Valid: true},
// 		)
// 		if err != nil {
// 			log.Printf("Failed to count all borrows for user %v: %v", user.ID, err)
// 			allBorrowsCount = 0
// 		}

// 		response = append(response, models.UserResponse{
// 			ID:                 user.ID.Bytes,
// 			FirstName:          user.FirstName,
// 			LastName:           user.LastName,
// 			Bio:                user.Bio,
// 			Email:              user.Email,
// 			PhoneNumber:        user.PhoneNumber,
// 			Role:               user.Role.String,
// 			IsBanned:           user.IsBanned.Bool,
// 			BanUntil:           &user.BanUntil.Time,
// 			BanReason:          user.BanReason.String,
// 			IsPermanentBan:     user.IsPermanentBan.Bool,
// 			AllBorrowsCount:    int(allBorrowsCount),
// 			ActiveBorrowsCount: int(activeBorrowsCount),
// 			CreatedAt:          user.CreatedAt.Time,

// 			// 👈 optional: add field to struct
// 		})
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"page":        page,
// 		"limit":       limit,
// 		"count":       len(response),
// 		"total_count": totalCount,
// 		"total_pages": totalPages,
// 		"users":       response,
// 	})
// }

// func GetUserByIDHandler(c *gin.Context) {
// 	userIDVal, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "userID not found in context"})
// 		return
// 	}

// 	userUUID, ok := userIDVal.(uuid.UUID)
// 	if !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid userID type"})
// 		return
// 	}

// 	// Optional: check if the middleware set banned_user flag
// 	// bannedUserFlag, _ := c.Get("banned_user")

// 	// 2️⃣ Fetch user from DB
// 	user, err := db.Q.GetUserByID(c.Request.Context(), pgtype.UUID{Bytes: userUUID, Valid: true})
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
// 		return
// 	}

// 	// 3️⃣ Count borrows
// 	activeBorrowsCount, err := db.Q.CountActiveBorrowsByUserID(c.Request.Context(), pgtype.UUID{Bytes: user.ID.Bytes, Valid: true})
// 	if err != nil {
// 		log.Printf("Failed to count active borrows for user %v: %v", user.ID, err)
// 		activeBorrowsCount = 0
// 	}

// 	allBorrowsCount, err := db.Q.CountBorrowedBooksByUserID(c.Request.Context(), pgtype.UUID{Bytes: user.ID.Bytes, Valid: true})
// 	if err != nil {
// 		log.Printf("Failed to count all borrows for user %v: %v", user.ID, err)
// 		allBorrowsCount = 0
// 	}

// 	// 4️⃣ Build response
// 	resp := models.UserResponse{
// 		ID:                 user.ID.Bytes,
// 		FirstName:          user.FirstName,
// 		LastName:           user.LastName,
// 		Bio:                user.Bio,
// 		Email:              user.Email,
// 		PhoneNumber:        user.PhoneNumber,
// 		Role:               user.Role.String,
// 		IsBanned:           user.IsBanned.Bool,
// 		IsPermanentBan:     user.IsPermanentBan.Bool,
// 		BanReason:          user.BanReason.String,
// 		BanUntil:           &user.BanUntil.Time,
// 		AllBorrowsCount:    int(allBorrowsCount),
// 		ActiveBorrowsCount: int(activeBorrowsCount),
// 		CreatedAt:          user.CreatedAt.Time,
// 	}

// 	log.Printf("👤 Returning user data for user %v (banned: %v)", user.ID, user.IsBanned.Bool)
// 	c.JSON(http.StatusOK, resp)
// }
// func SearchUsersPaginatedHandler(c *gin.Context) {
// 	// Pagination
// 	page := 1
// 	limit := 10

// 	if p := c.Query("page"); p != "" {
// 		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
// 			page = parsed
// 		}
// 	}

// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
// 			limit = parsed
// 		}
// 	}

// 	offset := (page - 1) * limit

// 	// Search query (partial email)
// 	query := strings.TrimSpace(c.Query("email"))
// 	if query == "" {
// 		query = "" // or "*" depending on your query
// 	}

// 	log.Printf("🔍 Searching users: email='%s', page=%d", query, page)

// 	// SQLC params
// 	params := gen.SearchUsersByEmailWithPaginationParams{
// 		Column1: pgtype.Text{String: query, Valid: true}, // search term
// 		Limit:   int32(limit),
// 		Offset:  int32(offset),
// 	}

// 	// Execute search query
// 	rows, err := db.Q.SearchUsersByEmailWithPagination(c.Request.Context(), params)
// 	if err != nil {
// 		log.Printf("❌ Search error: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
// 		return
// 	}

// 	// Count total matching users
// 	totalCount, err := db.Q.CountUsersByEmail(c.Request.Context(), pgtype.Text{String: query, Valid: true})
// 	if err != nil {
// 		log.Printf("❌ Count error: %v", err)
// 		totalCount = 0
// 	}

// 	// Map response
// 	var result []models.UserResponse
// 	for _, user := range rows {
// 		result = append(result, models.UserResponse{
// 			ID:        user.ID.Bytes,
// 			FirstName: user.FirstName,
// 			LastName:  user.LastName,
// 			Email:     user.Email,
// 			Role:      user.Role.String,
// 			CreatedAt: user.CreatedAt.Time,
// 			// Add other fields if needed
// 		})
// 	}

// 	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

// 	c.JSON(http.StatusOK, gin.H{
// 		"page":        page,
// 		"limit":       limit,
// 		"count":       len(result),
// 		"total_count": totalCount,
// 		"total_pages": totalPages,
// 		"users":       result,
// 	})
// }

// // UpdateUserByIDHandler updates user by ID
// func UpdateUserByIDHandler(c *gin.Context) {
// 	// Parse UUID
// 	idStr := c.Param("id")
// 	parsedID, err := uuid.Parse(idStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
// 		return
// 	}

// 	// Fetch current user (to get old public_id)
// 	currentUser, err := db.Q.GetUserByID(c, pgtype.UUID{Bytes: parsedID, Valid: true})
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// 		return
// 	}

// 	// Parse the incoming request
// 	var req models.UpdateUserRequest
// 	contentType := c.ContentType()

// 	if strings.HasPrefix(contentType, "multipart/form-data") {
// 		err = c.ShouldBind(&req)
// 	} else {
// 		err = c.ShouldBindJSON(&req)
// 	}

// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	params := gen.UpdateUserByIDParams{
// 		ID: pgtype.UUID{Bytes: parsedID, Valid: true},
// 	}

// 	// ===========================
// 	// Simple field updates
// 	// ===========================

// 	if req.FirstName != nil {
// 		params.FirstName = pgtype.Text{String: *req.FirstName, Valid: true}
// 	}

// 	if req.LastName != nil {
// 		params.LastName = pgtype.Text{String: *req.LastName, Valid: true}
// 	}

// 	if req.PhoneNumber != nil {
// 		params.PhoneNumber = pgtype.Text{String: *req.PhoneNumber, Valid: true}
// 	}

// 	if req.Bio != nil {
// 		params.Bio = pgtype.Text{String: *req.Bio, Valid: true}
// 	}

// 	// ===========================
// 	// IMAGE HANDLING
// 	// ===========================

// 	var newImageURL string
// 	var newPublicID string

// 	if req.ProfileImg != nil {
// 		// 1) Delete old image if exists
// 		if currentUser.ProfileImgPublicID.Valid {
// 			_ = service.DeleteImageFromCloudinary(currentUser.ProfileImgPublicID.String)
// 		}

// 		// 2) Upload new image
// 		file, err := req.ProfileImg.Open()
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open image"})
// 			return
// 		}
// 		defer file.Close()

// 		newImageURL, newPublicID, err = service.UploadProfileImgToCloudinary(file, req.ProfileImg.Filename)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "image upload failed"})
// 			return
// 		}

// 		// 3) Set params
// 		params.ProfileImg = pgtype.Text{String: newImageURL, Valid: true}
// 		params.ProfileImgPublicID = pgtype.Text{String: newPublicID, Valid: true}
// 	}

// 	// Save changes
// 	updatedUser, err := db.Q.UpdateUserByID(c, params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
// 		return
// 	}

// 	// Response
// 	resp := models.UserResponse{
// 		ID:                updatedUser.ID.Bytes,
// 		FirstName:         updatedUser.FirstName,
// 		LastName:          updatedUser.LastName,
// 		Bio:               updatedUser.Bio,
// 		Email:             updatedUser.Email,
// 		PhoneNumber:       updatedUser.PhoneNumber,
// 		Role:              updatedUser.Role.String,
// 		ProfileImg:        &updatedUser.ProfileImg.String,
// 		IsBanned:          updatedUser.IsBanned.Bool,
// 		BanUntil:          &updatedUser.BanUntil.Time,
// 		BanReason:         updatedUser.BanReason.String,
// 		IsPermanentBan:    updatedUser.IsPermanentBan.Bool,
// 		CreatedAt:         updatedUser.CreatedAt.Time,
// 	}

// 	c.JSON(http.StatusOK, resp)
// }

// func DeleteProfileImage(c *gin.Context) {
// 	// Get userID from context (from auth middleware)
// 	userIDVal, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "userID not found in context"})
// 		return
// 	}

// 	userUUID, ok := userIDVal.(uuid.UUID)
// 	if !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid userID type"})
// 		return
// 	}

// 	// Fetch user to check existing image
// 	user, err := db.Q.GetUserByID(c, pgtype.UUID{
// 		Bytes: userUUID,
// 		Valid: true,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
// 		return
// 	}

// 	// If no profile image → return immediately
// 	if !user.ProfileImgPublicID.Valid || user.ProfileImgPublicID.String == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "no profile image to delete"})
// 		return
// 	}

// 	// Delete from Cloudinary
// 	err = service.DeleteImageFromCloudinary(user.ProfileImgPublicID.String)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete from cloudinary"})
// 		return
// 	}

// 	// Update DB → remove image + public_id
// 	params := gen.UpdateUserByIDParams{
// 		ID:                 pgtype.UUID{Bytes: userUUID, Valid: true},
// 		ProfileImg:         pgtype.Text{Valid: false}, // sets NULL
// 		ProfileImgPublicID: pgtype.Text{Valid: false}, // sets NULL
// 	}

// 	_, err = db.Q.UpdateUserByID(c, params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user record"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "profile image deleted successfully",
// 	})
// }

// func BanUserByIDHandler(c *gin.Context) {
// 	// Parse UUID
// 	idStr := c.Param("id")
// 	parsedID, err := uuid.Parse(idStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
// 		return
// 	}

// 	// Bind request
// 	var req models.BanRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Handle BanUntil
// 	var banUntil pgtype.Timestamp
// 	if req.IsPermanentBan {
// 		// Permanent ban has no expiration
// 		banUntil = pgtype.Timestamp{Valid: false}
// 	} else if req.BanUntil != nil {
// 		banUntil = pgtype.Timestamp{Time: *req.BanUntil, Valid: true}
// 	} else {
// 		banUntil = pgtype.Timestamp{Valid: false}
// 	}

// 	// Update user ban
// 	params := gen.UpdateUserBanByUserIDParams{
// 		ID:             pgtype.UUID{Bytes: parsedID, Valid: true},
// 		IsBanned:       pgtype.Bool{Bool: req.IsBanned, Valid: true},
// 		BanReason:      pgtype.Text{String: req.BanReason, Valid: true},
// 		BanUntil:       banUntil,
// 		IsPermanentBan: pgtype.Bool{Bool: req.IsPermanentBan, Valid: true},
// 	}

// 	updatedUser, err := db.Q.UpdateUserBanByUserID(c.Request.Context(), params)
// 	if err != nil {
// 		log.Printf("BanUser error: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user ban status"})
// 		return
// 	}

// 	// Prepare response
// 	var banUntilPtr *time.Time
// 	if updatedUser.BanUntil.Valid {
// 		banUntilPtr = &updatedUser.BanUntil.Time
// 	} else {
// 		banUntilPtr = nil
// 	}

// 	resp := models.UserResponse{
// 		ID:             updatedUser.ID.Bytes,
// 		UserName:      updatedUser.UserName,
// 		Bio:            updatedUser.Bio,
// 		Email:          updatedUser.Email,
// 		PhoneNumber:    updatedUser.PhoneNumber,
// 		CreatedAt:      updatedUser.CreatedAt.Time,
// 		Role:           updatedUser.Role.String,
// 		TokenVersion:   int(updatedUser.TokenVersion),
// 		IsBanned:       updatedUser.IsBanned.Bool,
// 		BanReason:      updatedUser.BanReason.String,
// 		BanUntil:       banUntilPtr, // properly handle null
// 		IsPermanentBan: updatedUser.IsPermanentBan.Bool,
// 	}

// 	c.JSON(http.StatusOK, resp)
// }
