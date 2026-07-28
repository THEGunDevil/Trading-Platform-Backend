package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
)

type PublicHandler struct {
    Queries *gen.Queries
}

func NewPublicHandler(queries *gen.Queries) *PublicHandler {
    return &PublicHandler{Queries: queries}
}

// GET /public/deposit-address
func (h *PublicHandler) GetDepositAddress(c *gin.Context) {
    value, err := h.Queries.GetPlatformSetting(c.Request.Context(), "deposit_address")
    if err != nil || value == "" {
        c.JSON(http.StatusOK, gin.H{"address": ""})
        return
    }
    c.JSON(http.StatusOK, gin.H{"address": value})
}