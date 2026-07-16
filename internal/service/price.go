// internal/service/price.go
package service

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

var priceCache = &PriceCache{
    cache: make(map[string]cachedPrice),
}

type PriceCache struct {
    mu    sync.RWMutex
    cache map[string]cachedPrice
}

type cachedPrice struct {
    Price     float64
    Timestamp time.Time
}

func GetCurrentPrice(symbol string) (float64, error) {
    // Check cache first
    priceCache.mu.RLock()
    if cached, ok := priceCache.cache[symbol]; ok {
        if time.Since(cached.Timestamp) < 10*time.Second {
            priceCache.mu.RUnlock()
            return cached.Price, nil
        }
    }
    priceCache.mu.RUnlock()
    
    // Fetch from Binance
    url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol)
    
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return 0, fmt.Errorf("failed to fetch price: %w", err)
    }
    defer resp.Body.Close()
    
    var result struct {
        Price string `json:"price"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }
    
    var price float64
    fmt.Sscanf(result.Price, "%f", &price)
    
    // Update cache
    priceCache.mu.Lock()
    priceCache.cache[symbol] = cachedPrice{
        Price:     price,
        Timestamp: time.Now(),
    }
    priceCache.mu.Unlock()
    
    return price, nil
}