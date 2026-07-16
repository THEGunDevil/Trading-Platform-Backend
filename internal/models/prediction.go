// internal/models/prediction.go
package models

import (
	"math"
	"math/big"
	"time"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PredictionRequest struct {
	CoinID          string  `json:"coin_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,min=1,max=5000"`
	Direction       string  `json:"direction" binding:"required,oneof=up down"`
	DurationSeconds int     `json:"duration_seconds" binding:"required,oneof=10 30 60 300"`
}

type PredictionResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	CoinID          string     `json:"coin_id"`
	Symbol          string     `json:"symbol"`
	Amount          float64    `json:"amount"`
	Direction       string     `json:"direction"`
	DurationSeconds int        `json:"duration_seconds"`
	StartPrice      float64    `json:"start_price"`
	FinalPrice      *float64   `json:"final_price,omitempty"`
	PayoutRate      float64    `json:"payout_rate"`
	Status          string     `json:"status"`
	Profit          float64    `json:"profit"`
	Payout          float64    `json:"payout"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

type PredictionResult struct {
	Status     string  `json:"status"`
	Profit     float64 `json:"profit"`
	Payout     float64 `json:"payout"`
	FinalPrice float64 `json:"final_price"`
	StartPrice float64 `json:"start_price"`
}

type PredictionHistory struct {
	Predictions []PredictionResponse `json:"predictions"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	Limit       int                  `json:"limit"`
}

func convertNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}

	// Convert big.Int to float64
	f := new(big.Float).SetInt(n.Int)
	if n.Exp != 0 {
		// Apply exponent
		exp := new(big.Float).SetFloat64(math.Pow10(int(-n.Exp)))
		f.Mul(f, exp)
	}

	result, _ := f.Float64()
	return result
}
func ToPredictionResponse(p gen.Prediction) PredictionResponse {
	var finalPrice *float64
	if p.FinalPrice.Valid {
		f := convertNumericToFloat64(p.FinalPrice)
		finalPrice = &f
	}

	var resolvedAt *time.Time
	if p.ResolvedAt.Valid {
		t := p.ResolvedAt.Time
		resolvedAt = &t
	}

	return PredictionResponse{
		ID:              p.ID.Bytes,
		UserID:          p.UserID.Bytes,
		CoinID:          p.CoinID,
		Symbol:          p.Symbol,
		Amount:          convertNumericToFloat64(p.Amount),
		Direction:       p.Direction,
		DurationSeconds: int(p.DurationSeconds),
		StartPrice:      convertNumericToFloat64(p.StartPrice),
		FinalPrice:      finalPrice,
		PayoutRate:      convertNumericToFloat64(p.PayoutRate),
		Status:          p.Status,
		Profit:          convertNumericToFloat64(p.Profit),
		Payout:          convertNumericToFloat64(p.Payout),
		CreatedAt:       p.CreatedAt.Time,
		ExpiresAt:       p.ExpiresAt.Time,
		ResolvedAt:      resolvedAt,
	}
}
