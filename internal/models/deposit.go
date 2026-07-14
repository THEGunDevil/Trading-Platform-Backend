package models

import (
	"time"

	"github.com/google/uuid"
)

type Deposit struct {
	ID            uuid.UUID  `json:"id"`
	Asset         string     `json:"asset"`
	Network       string     `json:"network"`
	Amount        string     `json:"amount"`
	TxHash        *string    `json:"tx_hash,omitempty"`
	Confirmations int32      `json:"confirmations"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

type DepositAddressResponse struct {
	Asset   string `json:"asset"`
	Network string `json:"network"`
	Address string `json:"address"`
}