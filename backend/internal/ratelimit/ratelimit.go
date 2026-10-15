// Package ratelimit implements a simple, in-memory, per-key rate limiter
// (section 28: "Rate limiting where appropriate"). It's applied to the
// login endpoint specifically — brute-force password guessing is the
// concrete risk there — not blanket across every route; section 39 is
// explicit about not over-engineering security infrastructure, and most
// routes here are already gated by auth + RBAC, which a rate limit
// wouldn't add much to.
//
// This is a single-process, in-memory limiter: correct and sufficient for
// this project's single-backend-instance deployment target (section 4:
// "Keep the deployment architecture simple"). It would need a shared
// store (Redis, etc.) to stay correct behind multiple replicas — flagging
// that rather than building for a scale this project doesn't target.
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

type bucket struct {
	count     int
	windowEnd time.Time
}

// Limiter is a fixed-window, per-key request counter: each key gets up to
// `limit` requests per `window`; the window resets (not slides) on expiry.
// A fixed window is simpler than a sliding one and permits a short burst
// right at the window boundary — an acceptable trade for a login endpoint,
// where the goal is blunting sustained brute force, not exact precision.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
}

// Allow reports whether the request identified by key is within the rate
// limit, incrementing its counter as a side effect.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Opportunistic cleanup instead of a background goroutine — simple and
	// sufficient at this project's scale (section 39). Only sweeps once the
	// map has grown large enough for it to matter.
	if len(l.buckets) > 1000 {
		for k, b := range l.buckets {
			if now.After(b.windowEnd) {
				delete(l.buckets, k)
			}
		}
	}

	b, exists := l.buckets[key]
	if !exists || now.After(b.windowEnd) {
		l.buckets[key] = &bucket{count: 1, windowEnd: now.Add(l.window)}
		return true
	}

	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// Middleware returns Gin middleware that rate-limits by client IP,
// responding 429 in the same structured error shape as every other
// rejection in this API.
func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			apierror.Respond(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many attempts. Please try again later.", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
