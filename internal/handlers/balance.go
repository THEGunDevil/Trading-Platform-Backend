package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/internal/db"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	asset := c.Param("asset")

	balance, err := db.Q.GetBalance(c.Request.Context(), gen.GetBalanceParams{
		UserID: service.UUIDToPGType(userID),
		Asset:  asset,
	})

	if err != nil {
		// If the record doesn't exist, return a 200 OK with a 0 balance inline
		// Note: You can add an explicit check for pgx.ErrNoRows / sql.ErrNoRows here
		// if you want to differentiate from actual database connection errors
		c.JSON(http.StatusOK, gin.H{
			"asset":   asset,
			"balance": "0.00", // Adjust type (string/int/float) to match your frontend model
		})
		return
	}

	// Happy path: Record exists
	result, err := service.ToBalanceModel(balance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to format balance"})
		return
	}

	c.JSON(http.StatusOK, result)
}
