package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

// GET /api/balances/
func ListBalances(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	balances, err := db.Q.ListBalances(c.Request.Context(), service.UUIDToPGType(userID))
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to load balances")
		return
	}

	result, err := service.ToBalanceModels(balances)
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to format balances")
		return
	}

	service.WriteJSON(c, http.StatusOK, result)

}

// GET /api/balances/:asset
func GetBalance(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	asset := c.Param("asset")

	balance, err := db.Q.GetBalance(c.Request.Context(), gen.GetBalanceParams{
		UserID: service.UUIDToPGType(userID),
		Asset:  asset,
	})
	if err != nil {
		service.AbortWithError(c, http.StatusNotFound, "balance not found")
		return
	}

	result, err := service.ToBalanceModel(balance)
	if err != nil {
		service.AbortWithError(c, http.StatusInternalServerError, "failed to format balance")
		return
	}

	service.WriteJSON(c, http.StatusOK, result)
}
