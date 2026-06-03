package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for range time.Tick(5 * time.Minute) {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for ip, times := range rl.attempts {
			filtered := []time.Time{}
			for _, t := range times {
				if t.After(cutoff) {
					filtered = append(filtered, t)
				}
			}
			if len(filtered) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = filtered
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	filtered := []time.Time{}
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= rl.max {
		rl.attempts[ip] = filtered
		return false
	}

	rl.attempts[ip] = append(filtered, now)
	return true
}

var loginLimiter = newRateLimiter(5, 15*time.Minute)

func RateLimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !loginLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de tentatives, réessaie dans 15 minutes",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}