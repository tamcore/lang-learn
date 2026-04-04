package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	t.Parallel()
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     3,
		window:   time.Minute,
	}

	// First 3 requests should be allowed
	assert.True(t, rl.allow("1.2.3.4"))
	assert.True(t, rl.allow("1.2.3.4"))
	assert.True(t, rl.allow("1.2.3.4"))

	// 4th should be blocked
	assert.False(t, rl.allow("1.2.3.4"))
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	t.Parallel()
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     2,
		window:   time.Minute,
	}

	// Exhaust IP-A
	assert.True(t, rl.allow("ip-a"))
	assert.True(t, rl.allow("ip-a"))
	assert.False(t, rl.allow("ip-a"))

	// IP-B should still be allowed
	assert.True(t, rl.allow("ip-b"))
	assert.True(t, rl.allow("ip-b"))
	assert.False(t, rl.allow("ip-b"))
}

func TestRateLimiter_WindowReset(t *testing.T) {
	t.Parallel()
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     2,
		window:   50 * time.Millisecond,
	}

	assert.True(t, rl.allow("1.2.3.4"))
	assert.True(t, rl.allow("1.2.3.4"))
	assert.False(t, rl.allow("1.2.3.4"))

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again after window reset
	assert.True(t, rl.allow("1.2.3.4"))
	assert.True(t, rl.allow("1.2.3.4"))
	assert.False(t, rl.allow("1.2.3.4"))
}

func TestRateLimit_Middleware_AllowsRequests(t *testing.T) {
	t.Parallel()
	mw := RateLimit(5, time.Minute)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_Middleware_BlocksAfterLimit(t *testing.T) {
	t.Parallel()
	mw := RateLimit(2, time.Minute)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "rate limit exceeded")
}

func TestRateLimit_Middleware_DifferentIPsIndependent(t *testing.T) {
	t.Parallel()
	mw := RateLimit(1, time.Minute)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP-A uses its one request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// IP-A blocked
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// IP-B still allowed
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.4:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
