package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
    "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
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
// GET /admin/withdrawals/completed
func (h *AdminWithdrawalHandler) ListCompletedWithdrawals(c *gin.Context) {
    withdrawals, err := h.Queries.ListCompletedWithdrawals(c.Request.Context())
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch completed withdrawals")
        return
    }
    c.JSON(http.StatusOK, withdrawals)
}

// GET /admin/withdrawals/rejected
func (h *AdminWithdrawalHandler) ListRejectedWithdrawals(c *gin.Context) {
    withdrawals, err := h.Queries.ListRejectedWithdrawals(c.Request.Context())
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch rejected withdrawals")
        return
    }
    c.JSON(http.StatusOK, withdrawals)
}