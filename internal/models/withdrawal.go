package models

import (
	"time"

	"github.com/google/uuid"
)

type CreateWithdrawalRequest struct {
	Asset               string `json:"asset" validate:"required"`
	Network             string `json:"network" validate:"required"`
	DestinationAddress  string `json:"destination_address" validate:"required"`
	Amount              string `json:"amount" validate:"required"`
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