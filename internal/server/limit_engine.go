package server

import (
	"context"
	"log"

	"github.com/internal/service"
)

// StartLimitOrderEngine initializes and returns the WebSocket-based limit order monitor
func StartLimitOrderEngine(orderService *service.OrderService) *service.LimitOrderEngine {
	engine := service.NewLimitOrderEngine(orderService)

	// Load existing open orders
	if err := engine.LoadOpenOrders(context.Background()); err != nil {
		log.Printf("⚠️ Failed to load initial open orders: %v", err)
	}

	// Start WebSocket monitor in background
	go engine.StartWebSocketMonitor()

	log.Println("✅ Limit Order Engine started successfully")
	return engine
}
