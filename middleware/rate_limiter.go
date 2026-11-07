package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients       = make(map[string]*client)
	clientsMu     sync.Mutex
	cleanupOnce   sync.Once
	nextLimiterID atomic.Uint64
)

// RateLimiter creates an independent token bucket for this middleware instance.
// rateLimit is requests per second and burst is the maximum immediate burst.
func RateLimiter(rateLimit rate.Limit, burst int) gin.HandlerFunc {
	if rateLimit <= 0 {
		rateLimit = rate.Limit(1)
	}
	if burst < 1 {
		burst = 1
	}

	limiterID := nextLimiterID.Add(1)
	cleanupOnce.Do(func() { go cleanupClients() })

	return func(c *gin.Context) {
		identity := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(string); ok && id != "" {
				identity = "user:" + id
			}
		}
		key := strconv.FormatUint(limiterID, 10) + ":" + identity

		clientsMu.Lock()
		cl, exists := clients[key]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(rateLimit, burst)}
			clients[key] = cl
		}
		cl.lastSeen = time.Now()
		clientsMu.Unlock()

		if !cl.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Slow down.",
			})
			return
		}
		c.Next()
	}
}

func cleanupClients() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		clientsMu.Lock()
		for key, cl := range clients {
			if cl.lastSeen.Before(cutoff) {
				delete(clients, key)
			}
		}
		clientsMu.Unlock()
	}
}
