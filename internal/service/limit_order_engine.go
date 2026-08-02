package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	gen "github.com/internal/db/gen"
)

type LimitOrderEngine struct {
	orderSvc   *OrderService
	openOrders map[string][]LimitOrderCache
	inFlight   map[string]bool // orderID currently being executed, guarded by mu
	mu         sync.RWMutex
}

type LimitOrderCache struct {
	ID    string
	Side  string
	Price float64
}

func NewLimitOrderEngine(orderSvc *OrderService) *LimitOrderEngine {
	return &LimitOrderEngine{
		orderSvc:   orderSvc,
		openOrders: make(map[string][]LimitOrderCache),
		inFlight:   make(map[string]bool),
	}
}

// AddOrder adds a new limit order to the cache.
func (e *LimitOrderEngine) AddOrder(order gen.Order) {
	e.mu.Lock()
	defer e.mu.Unlock()

	orderUUID, _ := uuid.FromBytes(order.ID.Bytes[:])
	orderID := orderUUID.String()
	price := NumericToFloat64(order.Price)
	symbol := strings.ToUpper(order.Symbol)

	e.openOrders[symbol] = append(e.openOrders[symbol], LimitOrderCache{
		ID:    orderID,
		Side:  order.Side,
		Price: price,
	})
	log.Printf("➕ Added to Limit Engine cache: %s %s %s @ %.4f", symbol, order.Side, orderID, price)
}

// RemoveOrder removes an order from the cache (by symbol and ID).
func (e *LimitOrderEngine) RemoveOrder(symbol, orderID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	symbol = strings.ToUpper(symbol)
	orders := e.openOrders[symbol]
	for i, o := range orders {
		if o.ID == orderID {
			e.openOrders[symbol] = append(orders[:i], orders[i+1:]...)
			log.Printf("🗑️ Removed %s from memory cache. Remaining for %s: %d", orderID, symbol, len(e.openOrders[symbol]))
			break
		}
	}
	delete(e.inFlight, orderID)
}

// LoadOpenOrders reloads all open limit orders from the database.
func (e *LimitOrderEngine) LoadOpenOrders(ctx context.Context) error {
	orders, err := e.orderSvc.GetOpenLimitOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to load open orders: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	rebuilt := make(map[string][]LimitOrderCache)

	for _, order := range orders {
		orderUUID, _ := uuid.FromBytes(order.ID.Bytes[:])
		orderID := orderUUID.String()

		// Skip anything mid-execution from a trigger — otherwise a reload
		// landing mid-fill hands it back out and a second price tick can
		// trigger it again before the first attempt finishes.
		if e.inFlight[orderID] {
			continue
		}

		price := NumericToFloat64(order.Price)
		symbol := strings.ToUpper(order.Symbol)

		rebuilt[symbol] = append(rebuilt[symbol], LimitOrderCache{
			ID:    orderID,
			Side:  order.Side,
			Price: price,
		})
	}

	e.openOrders = rebuilt
	return nil
}

// StartWebSocketMonitor runs the WebSocket price feed and triggers execution.
func (e *LimitOrderEngine) StartWebSocketMonitor() {
	if err := e.LoadOpenOrders(context.Background()); err != nil {
		log.Printf("❌ Initial LoadOpenOrders failed: %v", err)
	}

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := e.LoadOpenOrders(context.Background()); err != nil {
				log.Printf("❌ Periodic LoadOpenOrders failed: %v", err)
			}
		}
	}()

	// !ticker@arr was deprecated by Binance (Nov 2025) and can be pulled with
	// no further notice. !miniTicker@arr is the supported replacement and
	// still has the "s" (symbol) and "c" (last price) fields this needs.
	url := "wss://stream.binance.com:9443/ws/!miniTicker@arr"

	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("❌ Connection failed: %v — retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("✅ Connected to Binance WS (all mini tickers)...")

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("⚠️ WS read error: %v — reconnecting...", err)
				conn.Close()
				break
			}

			var tickers []struct {
				Symbol string `json:"s"`
				Price  string `json:"c"`
			}
			if err := json.Unmarshal(message, &tickers); err != nil {
				log.Printf("⚠️ Failed to unmarshal ticker payload: %v", err)
				continue
			}

			for _, ticker := range tickers {
				currentPrice, err := strconv.ParseFloat(ticker.Price, 64)
				if err != nil || currentPrice <= 0 {
					continue
				}
				symbol := strings.ToUpper(ticker.Symbol)

				var triggeredOrders []LimitOrderCache

				e.mu.Lock()
				if orders, exists := e.openOrders[symbol]; exists && len(orders) > 0 {
					var remaining []LimitOrderCache
					for _, order := range orders {
						shouldExecute := false
						if order.Side == "buy" && currentPrice <= order.Price {
							shouldExecute = true
						} else if order.Side == "sell" && currentPrice >= order.Price {
							shouldExecute = true
						}

						if shouldExecute {
							triggeredOrders = append(triggeredOrders, order)
							e.inFlight[order.ID] = true
						} else {
							remaining = append(remaining, order)
						}
					}
					e.openOrders[symbol] = remaining
				}
				e.mu.Unlock()

				for _, order := range triggeredOrders {
					go func(orderID string, execPrice float64) {
						parsedID, _ := uuid.Parse(orderID)
						err := e.orderSvc.ExecuteLimitOrder(context.Background(), parsedID, execPrice)
						if err != nil {
							log.Printf("❌ Execute failed for %s: %v", orderID, err)
							e.mu.Lock()
							delete(e.inFlight, orderID)
							e.mu.Unlock()
						} else {
							log.Printf("✅ SUCCESS: %s executed at $%.4f", orderID, execPrice)
						}
					}(order.ID, currentPrice)
				}
			}
		}

		time.Sleep(3 * time.Second)
	}
}
