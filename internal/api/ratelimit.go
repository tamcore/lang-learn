package api

import (
	"net/http"
	"sync"
	"time"
)

// rateLimiter implements a simple per-IP token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // tokens per window
	window   time.Duration // window size
}

type visitor struct {
	tokens    int
	lastReset time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	// Clean up stale entries periodically.
	go func() {
		for range time.Tick(window * 2) {
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastReset) > window*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	now := time.Now()

	if !ok {
		rl.visitors[ip] = &visitor{tokens: rl.rate - 1, lastReset: now}
		return true
	}

	if now.Sub(v.lastReset) > rl.window {
		v.tokens = rl.rate - 1
		v.lastReset = now
		return true
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}
	return false
}

// RateLimit returns middleware that limits requests per IP.
func RateLimit(rate int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(rate, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(r.RemoteAddr) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
