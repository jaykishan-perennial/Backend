package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	rateLimitCleanupInterval = 1 * time.Minute
	rateLimitStaleAfter      = 10 * time.Minute
)

type ipLimiter struct {
	mu      sync.Mutex
	rps     float64
	burst   float64
	buckets map[string]*bucketEntry
}

type bucketEntry struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

func RateLimiter(rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 || burst <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	burstF := float64(burst)
	lim := &ipLimiter{
		rps:     rps,
		burst:   burstF,
		buckets: make(map[string]*bucketEntry),
	}

	go lim.cleanupLoop()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		lim.mu.Lock()
		b, ok := lim.buckets[ip]
		if !ok {
			b = &bucketEntry{
				tokens:   burstF,
				last:     now,
				lastSeen: now,
			}
			lim.buckets[ip] = b
		}
		b.lastSeen = now

		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = minFloat(lim.burst, b.tokens+elapsed*lim.rps)
			b.last = now
		}

		if b.tokens >= 1 {
			b.tokens--
			lim.mu.Unlock()
			c.Next()
			return
		}

		lim.mu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "Too many requests",
		})
		c.Abort()
	}
}

func (l *ipLimiter) cleanupLoop() {
	ticker := time.NewTicker(rateLimitCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.mu.Lock()
		for ip, b := range l.buckets {
			if now.Sub(b.lastSeen) >= rateLimitStaleAfter {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
