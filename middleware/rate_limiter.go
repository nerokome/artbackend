package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*client)
	mu      sync.Mutex
)

// RateLimiter creates a per-IP rate limiter middleware.
// rateLimit = requests per second
// burst = max burst allowed
func RateLimiter(rateLimit rate.Limit, burst int) gin.HandlerFunc {
	// Start cleanup goroutine once
	go cleanupClients()

	return func(c *gin.Context) {
		ip := getClientIP(c)

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			cl = &client{
				limiter:  rate.NewLimiter(rateLimit, burst),
				lastSeen: time.Now(),
			}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		if !cl.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Slow down.",
			})
			return
		}

		c.Next()
	}
}

// getClientIP checks X-Forwarded-For header first, then falls back to ClientIP
func getClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		// If multiple IPs in header, take the first one
		ip = strings.Split(ip, ",")[0]
		ip = strings.TrimSpace(ip)
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}


func cleanupClients() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, cl := range clients {
			if time.Since(cl.lastSeen) > 10*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}
}
