// internal/handlers/prediction.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// PlacePrediction handles creating a new prediction
func PlacePrediction(c *gin.Context) {
	// Get user ID using your helper
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.PredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		service.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}

	userIDPg := service.UUIDToPGType(userID)

	// Check active predictions limit (max 5)
	activeCount, err := db.Q.CountActivePredictionsByUser(c.Request.Context(), userIDPg)
	if err != nil {
		log.Printf("Failed to count active predictions: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to check predictions")
		return
	}

	if activeCount >= 5 {
		service.WriteError(c, http.StatusBadRequest, "Maximum 5 active predictions allowed")
		return
	}

	// Get user's USDT balance
	balance, err := db.Q.GetUserBalance(c.Request.Context(), userIDPg)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to check balance")
		return
	}

	if balance < req.Amount {
		service.WriteError(c, http.StatusBadRequest, "Insufficient balance")
		return
	}

	// Get current price from Binance
	symbol := service.CoinIDToSymbol(req.CoinID)
	currentPrice, err := service.GetCurrentPrice(symbol)
	if err != nil {
		log.Printf("Failed to get price for %s: %v", symbol, err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to get current price")
		return
	}
	currentPriceNumeric, err := service.Float64ToNumeric(currentPrice)
	if err != nil {
		log.Printf("Failed to convert current price: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Invalid price")
		return
	}

	// Lock funds (deduct from available)
	amountNumeric, err := service.Float64ToNumeric(req.Amount)
	if err != nil {
		log.Printf("Failed to convert amount: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Invalid amount")
		return
	}

	_, err = db.Q.LockBalance(c.Request.Context(), gen.LockBalanceParams{
		UserID:    userIDPg,
		Asset:     "USDT",
		Available: amountNumeric,
	})
	if err != nil {
		log.Printf("Failed to lock balance: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Insufficient balance")
		return
	}
	payoutRate, err := service.Float64ToNumeric(0.8000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to convert payout rate"})
		return
	}

	// Create prediction
	prediction, err := db.Q.CreatePrediction(c.Request.Context(), gen.CreatePredictionParams{
		UserID:          userIDPg,
		CoinID:          req.CoinID,
		Symbol:          symbol,
		Amount:          amountNumeric,
		Direction:       req.Direction,
		DurationSeconds: int32(req.DurationSeconds),
		StartPrice:      currentPriceNumeric,
		PayoutRate:      payoutRate,
	})

	if err != nil {
		// Rollback: unlock balance
		db.Q.UnlockBalance(c.Request.Context(), gen.UnlockBalanceParams{
			UserID: userIDPg,
			Asset:  "USDT",
			Locked: amountNumeric,
		})
		log.Printf("Failed to create prediction: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to create prediction")
		return
	}

	// Schedule auto-resolution in background
	go schedulePredictionResolution(prediction.ID, time.Duration(req.DurationSeconds)*time.Second)

	resp := models.ToPredictionResponse(prediction)
	service.WriteJSON(c, http.StatusCreated, resp)
}

// GetPredictionResult returns the result of a prediction
func GetPredictionResult(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	predictionID, err := service.ParseUUIDParam(c, "id")
	if err != nil {
		service.WriteError(c, http.StatusBadRequest, "Invalid prediction ID")
		return
	}

	prediction, err := db.Q.GetPredictionByID(c.Request.Context(), gen.GetPredictionByIDParams{
		ID:     service.UUIDToPGType(predictionID),
		UserID: service.UUIDToPGType(userID),
	})

	if err != nil {
		service.WriteError(c, http.StatusNotFound, "Prediction not found")
		return
	}

	resp := models.ToPredictionResponse(prediction)
	service.WriteJSON(c, http.StatusOK, resp)
}

// GetActivePredictions returns user's active predictions
func GetActivePredictions(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	predictions, err := db.Q.GetActivePredictionsByUser(c.Request.Context(),
		service.UUIDToPGType(userID))
	if err != nil {
		log.Printf("Failed to get active predictions: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to fetch predictions")
		return
	}

	var resp []models.PredictionResponse
	for _, p := range predictions {
		resp = append(resp, models.ToPredictionResponse(p))
	}

	service.WriteJSON(c, http.StatusOK, gin.H{"predictions": resp})
}

// GetPredictionHistory returns user's prediction history with pagination
func GetPredictionHistory(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	predictions, err := db.Q.GetUserPredictions(c.Request.Context(), gen.GetUserPredictionsParams{
		UserID: service.UUIDToPGType(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		log.Printf("Failed to get prediction history: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to fetch predictions")
		return
	}

	var resp []models.PredictionResponse
	for _, p := range predictions {
		resp = append(resp, models.ToPredictionResponse(p))
	}

	service.WriteJSON(c, http.StatusOK, gin.H{
		"predictions": resp,
		"page":        page,
		"limit":       limit,
	})
}

// CancelPrediction cancels an active prediction with 50% refund
func CancelPrediction(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	predictionID, err := service.ParseUUIDParam(c, "id")
	if err != nil {
		service.WriteError(c, http.StatusBadRequest, "Invalid prediction ID")
		return
	}

	userIDPg := service.UUIDToPGType(userID)
	predictionIDPg := service.UUIDToPGType(predictionID)

	// Get prediction
	prediction, err := db.Q.GetPredictionByID(c.Request.Context(), gen.GetPredictionByIDParams{
		ID:     predictionIDPg,
		UserID: userIDPg,
	})

	if err != nil || prediction.Status != "active" {
		service.WriteError(c, http.StatusBadRequest, "Prediction not found or not active")
		return
	}

	// Calculate 50% refund
	amountFloat := service.NumericToFloat64(prediction.Amount)
	refundAmount := amountFloat * 0.5

	refundNumeric, err := service.Float64ToNumeric(refundAmount)
	if err != nil {
		service.WriteError(c, http.StatusInternalServerError, "Failed to calculate refund")
		return
	}

	// Unlock (refund) 50% of locked amount
	_, err = db.Q.UnlockBalance(c.Request.Context(), gen.UnlockBalanceParams{
		UserID: userIDPg,
		Asset:  "USDT",
		Locked: refundNumeric,
	})
	if err != nil {
		log.Printf("Failed to refund: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to process refund")
		return
	}

	// Cancel prediction
	err = db.Q.CancelPrediction(c, gen.CancelPredictionParams{
		ID:     predictionIDPg,
		UserID: userIDPg,
	})
	if err != nil {
		log.Printf("Failed to cancel prediction: %v", err)
		service.WriteError(c, http.StatusInternalServerError, "Failed to cancel prediction")
		return
	}

	service.WriteJSON(c, http.StatusOK, gin.H{
		"message":       "Prediction cancelled",
		"refund_amount": refundAmount,
	})
}

// schedulePredictionResolution runs in background goroutine
func schedulePredictionResolution(predictionID pgtype.UUID, duration time.Duration) {
	time.Sleep(duration)

	prediction, err := db.Q.GetPredictionByID(context.Background(), gen.GetPredictionByIDParams{
		ID: predictionID,
	})
	if err != nil || prediction.Status != "active" {
		return
	}

	// Get final price
	finalPrice, err := service.GetCurrentPrice(prediction.Symbol)
	if err != nil {
		log.Printf("Failed to get final price: %v", err)
		finalPrice = service.NumericToFloat64(prediction.StartPrice)
	}
	finalPriceNumeric, err := service.Float64ToNumeric(finalPrice)
	if err != nil {
		fmt.Errorf("invalid price: %w", err)
	}
	startPrice := service.NumericToFloat64(prediction.StartPrice)

	// Determine result
	isWin := false
	if prediction.Direction == "up" {
		isWin = finalPrice > startPrice
	} else {
		isWin = finalPrice < startPrice
	}

	status := "lost"
	if isWin {
		status = "won"
	}

	// Resolve prediction
	resolved, err := db.Q.ResolvePrediction(context.Background(), gen.ResolvePredictionParams{
		ID:         predictionID,
		Status:     status,
		FinalPrice: finalPriceNumeric,
	})

	if err != nil {
		log.Printf("Failed to resolve prediction: %v", err)
		return
	}

	// If won, credit payout to available balance
	if status == "won" {
		payoutFloat := service.NumericToFloat64(resolved.Payout)
		payoutNumeric, _ := service.Float64ToNumeric(payoutFloat)

		_, err = db.Q.IncreaseAvailableBalance(context.Background(), gen.IncreaseAvailableBalanceParams{
			UserID:    resolved.UserID,
			Asset:     "USDT",
			Available: payoutNumeric,
		})
		if err != nil {
			log.Printf("Failed to credit winnings: %v", err)
		}
	}
}

// CoinIDToSymbol converts internal coin ID to Binance symbol
