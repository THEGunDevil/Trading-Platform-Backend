package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	Requests int
	LastSeen time.Time
}

var (
	clients = make(map[string]*Client)
	mutex   sync.Mutex
	maxReq  = 100      // max requests per window
	window  = 1 * time.Minute // time window
)

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("skip_rate_limit") {
			c.Next()
			return
		}
		ip := c.ClientIP() // get client IP

		mutex.Lock()
		client, exists := clients[ip]

		// First request or window expired
		if !exists || time.Since(client.LastSeen) > window {
			clients[ip] = &Client{Requests: 1, LastSeen: time.Now()}
			mutex.Unlock()
			c.Next()
			return
		}

		// Exceeded limit
		if client.Requests >= maxReq {
			mutex.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		// Increment request count
		client.Requests++
		client.LastSeen = time.Now()
		mutex.Unlock()
		c.Next()
	}
}
func SkipRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("skip_rate_limit", true)
		c.Next()
	}
}
