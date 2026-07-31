package handlers

import (
	"net/http"

	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func (h *AdminSettingsHandler) UpdateWillProfitHandler(c *gin.Context) {
	idStr := c.Param("id")

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		WillProfit *bool `json:"will_profit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.WillProfit == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "will_profit is required"})
		return
	}

	params := gen.UpdateUserWillProfitParams{
		ID: pgtype.UUID{
			Bytes: parsedID,
			Valid: true,
		},
		WillProfit: pgtype.Bool{
			Bool:  *req.WillProfit,
			Valid: true,
		},
	}

	if err := h.Queries.UpdateUserWillProfit(c.Request.Context(), params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}
