package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter wraps a rate.Limiter with a last-seen timestamp used for cleanup.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipRateLimiter manages per-IP token bucket rate limiters.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	burst    int
}

// newIPRateLimiter creates a new ipRateLimiter.
// r is the sustained rate in requests per second; burst is the bucket capacity.
func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		burst:    burst,
	}
	// Start a background goroutine to evict stale entries every minute.
	go rl.cleanupLoop()
	return rl
}

func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(rl.r, rl.burst),
		}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Pre-built limiters used by the auth rate-limit middleware.
// login / 2fa:    5 requests per minute per IP  → ~0.0833 r/s, burst=5
// refresh:       10 requests per minute per IP → ~0.1667 r/s, burst=10
var (
	loginLimiter   = newIPRateLimiter(rate.Every(12*time.Second), 5) // 5/min
	refreshLimiter = newIPRateLimiter(rate.Every(6*time.Second), 10) // 10/min
)

// AuthRateLimit returns a Gin middleware that enforces per-IP rate limits on
// authentication endpoints. It distinguishes between the /refresh endpoint
// (higher limit) and all other auth endpoints (tighter limit).
func AuthRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		var limiter *rate.Limiter
		if c.FullPath() == "/api/v1/auth/refresh" {
			limiter = refreshLimiter.getLimiter(ip)
		} else {
			limiter = loginLimiter.getLimiter(ip)
		}

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}
