package models

import (
	"time"

	"github.com/google/uuid"
)

type PlaceOrderRequest struct {
	Symbol    string  `json:"symbol" validate:"required"`
	Side      string  `json:"side" validate:"required,oneof=buy sell"`
	OrderType string  `json:"order_type" validate:"required,oneof=market limit"`
	Leverage  int32   `json:"leverage" validate:"required,min=1,max=125"`
	Price     *string `json:"price,omitempty"` // required if order_type == limit, validated in service
	Quantity  string  `json:"quantity" validate:"required"`
}

type Order struct {
	ID        uuid.UUID  `json:"id"`
	Symbol    string     `json:"symbol"`
	Side      string     `json:"side"`
	OrderType string     `json:"order_type"`
	Leverage  int32      `json:"leverage"`
	Price     *string    `json:"price,omitempty"`
	Quantity  string     `json:"quantity"`
	Margin    string     `json:"margin"`
	Fee       string     `json:"fee"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	FilledAt  *time.Time `json:"filled_at,omitempty"`
}