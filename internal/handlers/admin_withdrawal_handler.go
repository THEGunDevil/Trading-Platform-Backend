package handlers

import (
	"math"
	"net/http"
	"strconv"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminWithdrawalHandler struct {
	Queries *gen.Queries
}

func NewAdminWithdrawalHandler(queries *gen.Queries) *AdminWithdrawalHandler {
	return &AdminWithdrawalHandler{Queries: queries}
}

// GET /admin/withdrawals/:search  (search by user id)
// GET /admin/withdrawals/search?q=john
func (h *AdminWithdrawalHandler) SearchWithdrawals(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		service.AbortWithError(c, http.StatusBadRequest, "search query is required")
		return
	}
	withdrawals, err := h.Queries.SearchWithdrawalsByUser(c.Request.Context(), service.StringToPGText(query))
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to search withdrawals")
		return
	}
	c.JSON(http.StatusOK, withdrawals)
}

// PATCH /admin/withdrawals/:id/approve
func (h *AdminWithdrawalHandler) ApproveWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid id")
		return
	}
	// Mark as completed (you can also attach a tx_hash if you accept it from request)
	_, err = h.Queries.MarkWithdrawalCompleted(c.Request.Context(), gen.MarkWithdrawalCompletedParams{
		ID:     service.UUIDToPGType(id),
		TxHash: service.StringToPGText(""), // or get from request
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to approve withdrawal")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"})
}

// PATCH /admin/withdrawals/:id/reject
func (h *AdminWithdrawalHandler) RejectWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid id")
		return
	}
	_, err = h.Queries.MarkWithdrawalRejected(c.Request.Context(), service.UUIDToPGType(id))
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to reject withdrawal")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "withdrawal rejected"})
}
func (h *AdminWithdrawalHandler) PendingWithdrawals(c *gin.Context) {
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

	pws, err := h.Queries.ListPendingWithdrawals(c.Request.Context(), gen.ListPendingWithdrawalsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch pending withdrawals")
		return
	}

	totalCount, err := h.Queries.CountPendingWithdrawals(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count users")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	// Build response
	response := make([]models.Withdrawal, len(pws))
	for i, pw := range pws {
		response[i] = models.ToWithdrawalResponse(pw)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(response),
		"total_count": totalCount,
		"total_pages": totalPages,
		"withdrawals": response,
	})
}

// GET /admin/withdrawals/completed
func (h *AdminWithdrawalHandler) CompletedWithdrawals(c *gin.Context) {
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

	pws, err := h.Queries.ListCompletedWithdrawals(c.Request.Context(), gen.ListCompletedWithdrawalsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch completed withdrawals")
		return
	}

	totalCount, err := h.Queries.CountCompletedWithdrawals(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count completed withdrawals")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	// Build response
	response := make([]models.Withdrawal, len(pws))
	for i, pw := range pws {
		response[i] = models.ToWithdrawalResponse(pw)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(response),
		"total_count": totalCount,
		"total_pages": totalPages,
		"withdrawals": response,
	})
}

// GET /admin/withdrawals/rejected
func (h *AdminWithdrawalHandler) RejectedWithdrawals(c *gin.Context) {
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

	pws, err := h.Queries.ListRejectedWithdrawals(c.Request.Context(), gen.ListRejectedWithdrawalsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch rejected withdrawals")
		return
	}

	totalCount, err := h.Queries.CountPendingWithdrawals(c.Request.Context())
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to count rejected withdrawals")
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	// Build response
	response := make([]models.Withdrawal, len(pws))
	for i, pw := range pws {
		response[i] = models.ToWithdrawalResponse(pw)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"count":       len(response),
		"total_count": totalCount,
		"total_pages": totalPages,
		"withdrawals": response,
	})
}