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

// GET /admin/withdrawals → returns all withdrawals (you can also use ListPendingWithdrawals if you prefer)
func (h *AdminWithdrawalHandler) ListAllWithdrawals(c *gin.Context) {
    // Using your existing ListPendingWithdrawals to get all pending; if you want all, you'd need a new query.
    // For now we'll just use the existing ListPendingWithdrawals, which is probably what you need.
    withdrawals, err := h.Queries.ListPendingWithdrawals(c.Request.Context())
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to fetch withdrawals")
        return
    }
    c.JSON(http.StatusOK, withdrawals)
}

// GET /admin/withdrawals/pending (optional, already covered by the above)
// You can have a separate one if needed.

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