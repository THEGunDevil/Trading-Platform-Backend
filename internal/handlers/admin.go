package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

// ---------- Dashboard ----------
type UserWithBalances struct {
	ID             uuid.UUID        `json:"id"`
	UserName       string           `json:"user_name"`
	Email          string           `json:"email"`
	IsBanned       bool             `json:"is_banned"`
	BanUntil       *time.Time       `json:"ban_until"`
	BanReason      string           `json:"ban_reason"`
	IsPermanentBan bool             `json:"is_permanent_ban"`
	CreatedAt      time.Time        `json:"created_at"`
	WillProfit     bool             `json:"will_profit"`
	Role           string           `json:"role"`
	Balances       []models.Balance `json:"balances"`
}

type AdminDashboardResponse struct {
	PlatformBalances       []gen.GetPlatformBalancesRow      `json:"balances"`
	Trades                 []gen.GetRecentTradesRow          `json:"trades"`
	Sessions               []gen.GetActiveSupportSessionsRow `json:"sessions"`
	PendingWithdrawals     []gen.Withdrawal                  `json:"pending_withdrawals"`
	PendingWithdrawalCount int                               `json:"pending_withdrawal_count"`
}
type AdminHandler struct {
	Queries *gen.Queries
}

func NewAdminHandler(queries *gen.Queries) *AdminHandler {
	return &AdminHandler{Queries: queries}
}

func (h *AdminHandler) GetAdminDashboard(c *gin.Context) {
	var (
		wg       sync.WaitGroup
		balances []gen.GetPlatformBalancesRow
		trades   []gen.GetRecentTradesRow
		sessions []gen.GetActiveSupportSessionsRow
		errs     []error
		mu       sync.Mutex
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		b, err := h.Queries.GetPlatformBalances(c.Request.Context())
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			balances = b
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		t, err := h.Queries.GetRecentTrades(c.Request.Context())
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			trades = t
		}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		s, err := h.Queries.GetActiveSupportSessions(c.Request.Context())
		mu.Lock()
		if err != nil {
			errs = append(errs, err)
		} else {
			sessions = s
		}
		mu.Unlock()
	}()

	wg.Wait()

	if len(errs) > 0 {
		log.Printf("Dashboard fetch errors: %v", errs)
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch dashboard data")
		return
	}

	c.JSON(http.StatusOK, AdminDashboardResponse{
		PlatformBalances: balances,
		Trades:           trades,
		Sessions:         sessions,
	})
}

// GET /admin/users
func (h *AdminHandler) GetUsersHandler(c *gin.Context) {
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

	users, err := h.Queries.ListUsersPaginated(c.Request.Context(), gen.ListUsersPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	totalCount, err := h.Queries.CountUsers(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count users")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// Collect user IDs for this page
	pgUUIDs := make([]pgtype.UUID, len(users))
	for i, user := range users {
		pgUUIDs[i] = pgtype.UUID{Bytes: user.ID.Bytes, Valid: true}
	}

	// Fetch balances for these users
	balanceMap := make(map[string][]models.Balance)
	if len(pgUUIDs) > 0 {
		balances, err := h.Queries.GetBalancesForUsers(c.Request.Context(), pgUUIDs)
		if err != nil {
			log.Printf("Failed to fetch balances for users: %v", err)
		} else {
			for _, b := range balances {
				// Convert [16]byte to string using uuid.UUID
				uidStr := uuid.UUID(b.UserID.Bytes).String()

				available, _ := service.NumericToString(b.Available)
				locked, _ := service.NumericToString(b.Locked)

				balanceMap[uidStr] = append(balanceMap[uidStr], models.Balance{
					UserID:    uuid.UUID(b.UserID.Bytes),
					Asset:     b.Asset,
					Available: available,
					Locked:    locked,
					UpdatedAt: time.Now(), // or b.UpdatedAt if you add it to the query
				})
			}
		}
	}

	// Build response
	response := make([]UserWithBalances, len(users))
	for i, user := range users {
		uidStr := uuid.UUID(user.ID.Bytes).String()
		response[i] = UserWithBalances{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Email:          user.Email,
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,
			Role:           user.Role.String,
			Balances:       balanceMap[uidStr],
			WillProfit:     user.WillProfit.Bool,
		}
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
func (h *AdminHandler) GetBannedUsersHandler(c *gin.Context) {
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

	users, err := h.Queries.ListBannedUsersPaginated(c.Request.Context(), gen.ListBannedUsersPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch banned users")
		return
	}

	totalCount, err := h.Queries.CountBannedUsers(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count banned users")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// Collect user IDs for this page
	pgUUIDs := make([]pgtype.UUID, len(users))
	for i, user := range users {
		pgUUIDs[i] = pgtype.UUID{Bytes: user.ID.Bytes, Valid: true}
	}

	// Fetch balances for these users
	balanceMap := make(map[string][]models.Balance)
	if len(pgUUIDs) > 0 {
		balances, err := h.Queries.GetBalancesForUsers(c.Request.Context(), pgUUIDs)
		if err != nil {
			log.Printf("Failed to fetch balances for banned users: %v", err)
		} else {
			for _, b := range balances {
				uidStr := uuid.UUID(b.UserID.Bytes).String()
				available, _ := service.NumericToString(b.Available)
				locked, _ := service.NumericToString(b.Locked)
				balanceMap[uidStr] = append(balanceMap[uidStr], models.Balance{
					UserID:    uuid.UUID(b.UserID.Bytes),
					Asset:     b.Asset,
					Available: available,
					Locked:    locked,
					UpdatedAt: time.Now(),
				})
			}
		}
	}

	// Build response
	response := make([]UserWithBalances, len(users))
	for i, user := range users {
		uidStr := uuid.UUID(user.ID.Bytes).String()
		response[i] = UserWithBalances{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Email:          user.Email,
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,
			Role:           user.Role.String,
			Balances:       balanceMap[uidStr],
		}
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

func (h *AdminHandler) GetAgentUsersHandler(c *gin.Context) {
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

	users, err := h.Queries.ListAgentUsersPaginated(c.Request.Context(), gen.ListAgentUsersPaginatedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch agent users")
		return
	}

	totalCount, err := h.Queries.CountAgentUsers(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count agent users")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// Collect user IDs for this page
	pgUUIDs := make([]pgtype.UUID, len(users))
	for i, user := range users {
		pgUUIDs[i] = pgtype.UUID{Bytes: user.ID.Bytes, Valid: true}
	}

	// Fetch balances for these users
	balanceMap := make(map[string][]models.Balance)
	if len(pgUUIDs) > 0 {
		balances, err := h.Queries.GetBalancesForUsers(c.Request.Context(), pgUUIDs)
		if err != nil {
			log.Printf("Failed to fetch balances for agent users: %v", err)
		} else {
			for _, b := range balances {
				uidStr := uuid.UUID(b.UserID.Bytes).String()
				available, _ := service.NumericToString(b.Available)
				locked, _ := service.NumericToString(b.Locked)
				balanceMap[uidStr] = append(balanceMap[uidStr], models.Balance{
					UserID:    uuid.UUID(b.UserID.Bytes),
					Asset:     b.Asset,
					Available: available,
					Locked:    locked,
					UpdatedAt: time.Now(),
				})
			}
		}
	}

	// Build response
	response := make([]UserWithBalances, len(users))
	for i, user := range users {
		uidStr := uuid.UUID(user.ID.Bytes).String()
		response[i] = UserWithBalances{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Email:          user.Email,
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,
			Role:           user.Role.String,
			Balances:       balanceMap[uidStr],
		}
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

// GET /admin/users/search?q=email
func (h *AdminHandler) SearchUsersHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		service.AbortWithError(c, http.StatusBadRequest, "search query is required")
		return
	}

	page := 1
	limit := 20

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

	users, err := h.Queries.SearchUsers(c.Request.Context(), gen.SearchUsersParams{
		Column1: service.StringToPGText(query), // ILIKE expects a plain string, not pgtype.Text
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to search users")
		return
	}

	// Collect user IDs
	pgUUIDs := make([]pgtype.UUID, len(users))
	for i, user := range users {
		pgUUIDs[i] = pgtype.UUID{Bytes: user.ID.Bytes, Valid: true}
	}

	// Fetch balances for these users
	balanceMap := make(map[string][]models.Balance)
	if len(pgUUIDs) > 0 {
		balances, err := h.Queries.GetBalancesForUsers(c.Request.Context(), pgUUIDs)
		if err != nil {
			log.Printf("Failed to fetch balances for search results: %v", err)
		} else {
			for _, b := range balances {
				uidStr := uuid.UUID(b.UserID.Bytes).String()
				available, _ := service.NumericToString(b.Available)
				locked, _ := service.NumericToString(b.Locked)
				balanceMap[uidStr] = append(balanceMap[uidStr], models.Balance{
					UserID:    uuid.UUID(b.UserID.Bytes),
					Asset:     b.Asset,
					Available: available,
					Locked:    locked,
					UpdatedAt: time.Now(),
				})
			}
		}
	}

	// Build response
	response := make([]UserWithBalances, len(users))
	for i, user := range users {
		uidStr := uuid.UUID(user.ID.Bytes).String()
		response[i] = UserWithBalances{
			ID:             user.ID.Bytes,
			UserName:       user.UserName,
			Email:          user.Email,
			IsBanned:       user.IsBanned.Bool,
			BanUntil:       &user.BanUntil.Time,
			BanReason:      user.BanReason.String,
			IsPermanentBan: user.IsPermanentBan.Bool,
			CreatedAt:      user.CreatedAt.Time,
			Role:           user.Role.String,
			Balances:       balanceMap[uidStr],
			WillProfit:     user.WillProfit.Bool,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": response,
	})
}

// PATCH /admin/users/ban/:id
func (h *AdminHandler) BanUser(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req struct {
		IsPermanent   bool   `json:"is_permanent"`
		Reason        string `json:"reason"`
		DurationHours *int   `json:"duration_hours,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var banUntil pgtype.Timestamp
	if req.IsPermanent {
		banUntil = pgtype.Timestamp{Valid: false}
	} else {
		duration := 24 // default 24 hours
		if req.DurationHours != nil {
			duration = *req.DurationHours
		}
		banUntil = pgtype.Timestamp{
			Time:  time.Now().Add(time.Duration(duration) * time.Hour),
			Valid: true,
		}
	}

	err = h.Queries.BanUser(c.Request.Context(), gen.BanUserParams{
		ID:             service.UUIDToPGType(parsedID),
		IsPermanentBan: pgtype.Bool{Bool: req.IsPermanent, Valid: true},
		BanReason:      service.StringToPGTextNullable(req.Reason),
		BanUntil:       banUntil,
	})
	if err != nil {
		log.Printf("BanUser error: %v", err)
		service.AbortWithError(c, http.StatusInternalServerError, "failed to ban user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user banned successfully"})
}

// PATCH /admin/users/unban/:id
func (h *AdminHandler) UnbanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	err = h.Queries.UnbanUser(c.Request.Context(), service.UUIDToPGType(userID))
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to unban user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user unbanned successfully"})
}

// PATCH /admin/users/role/:id
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "role is required")
		return
	}

	if req.Role != "user" && req.Role != "agent" && req.Role != "admin" {
		service.AbortWithError(c, http.StatusBadRequest, "invalid role")
		return
	}

	err = h.Queries.UpdateUserRole(c.Request.Context(), gen.UpdateUserRoleParams{
		ID:   service.UUIDToPGType(userID),
		Role: service.StringToPGTextNullable(req.Role),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to update role")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user role updated successfully"})
}

// POST /admin/deposit
func (h *AdminHandler) AdminDeposit(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Amount string `json:"amount" binding:"required"`
		Asset  string `json:"asset" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid request")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	amountNumeric, err := service.StringToNumeric(req.Amount)
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid amount")
		return
	}

	// Upsert balance (creates row if it doesn't exist, then adds)
	_, err = h.Queries.UpsertBalance(c.Request.Context(), gen.UpsertBalanceParams{
		UserID: service.UUIDToPGType(userID),
		Asset:  req.Asset,
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to upsert balance")
		return
	}

	_, err = h.Queries.IncreaseAvailableBalance(c.Request.Context(), gen.IncreaseAvailableBalanceParams{
		UserID:    service.UUIDToPGType(userID),
		Asset:     req.Asset,
		Available: amountNumeric,
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to deposit")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deposit successful"})
}
