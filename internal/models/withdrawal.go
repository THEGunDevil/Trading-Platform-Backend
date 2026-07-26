package models

import (
	"time"

	"github.com/google/uuid"
)

type CreateWithdrawalRequest struct {
    Asset              string `json:"asset" binding:"required"`
    Network            string `json:"network" binding:"required"`
    DestinationAddress string `json:"destination_address" binding:"required"`
    Amount             string `json:"amount" binding:"required"` // decimal string
    Fee                string `json:"fee" binding:"required"`
}

type Withdrawal struct {
	ID                 uuid.UUID  `json:"id"`
	Asset              string     `json:"asset"`
	Network            string     `json:"network"`
	DestinationAddress string     `json:"destination_address"`
	Amount             string     `json:"amount"`
	Fee                string     `json:"fee"`
	Status             string     `json:"status"`
	TxHash             *string    `json:"tx_hash,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
}
