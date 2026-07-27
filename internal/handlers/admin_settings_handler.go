package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
    "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

type AdminSettingsHandler struct {
    Queries *gen.Queries
}

func NewAdminSettingsHandler(queries *gen.Queries) *AdminSettingsHandler {
    return &AdminSettingsHandler{Queries: queries}
}

// GET /admin/settings/deposit-address
func (h *AdminSettingsHandler) GetDepositAddress(c *gin.Context) {
    value, err := h.Queries.GetPlatformSetting(c.Request.Context(), "deposit_address")
    if err != nil {
        // Return empty string if not set
        c.JSON(http.StatusOK, gin.H{"address": ""})
        return
    }
    c.JSON(http.StatusOK, gin.H{"address": value})
}

// PUT /admin/settings/deposit-address
func (h *AdminSettingsHandler) UpdateDepositAddress(c *gin.Context) {
    var req struct {
        Address string `json:"address" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        service.AbortWithError(c, http.StatusBadRequest, "address is required")
        return
    }
    err := h.Queries.UpsertPlatformSetting(c.Request.Context(), gen.UpsertPlatformSettingParams{
        Key:   "deposit_address",
        Value: req.Address,
    })
    if err != nil {
        service.AbortWithError(c, http.StatusInternalServerError, "failed to update deposit address")
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Deposit address updated"})
}