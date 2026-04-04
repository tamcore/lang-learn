package generator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLLMClient_DefaultModel(t *testing.T) {
	t.Parallel()
	c := NewLLMClient("my-key", "")
	assert.Equal(t, "google/gemini-2.5-flash", c.model)
	assert.Equal(t, "https://openrouter.ai/api/v1", c.baseURL)
	assert.Equal(t, "my-key", c.apiKey)
	assert.NotNil(t, c.client)
}

func TestNewLLMClient_CustomModel(t *testing.T) {
	t.Parallel()
	c := NewLLMClient("k", "anthropic/claude-3")
	assert.Equal(t, "anthropic/claude-3", c.model)
}

func TestLLMComplete_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req chatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "test-model", req.Model)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		assert.Equal(t, "say hello", req.Messages[0].Content)

		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "Hello!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewLLMClient("test-key", "test-model")
	c.baseURL = srv.URL

	result, err := c.Complete(context.Background(), "say hello")
	require.NoError(t, err)
	assert.Equal(t, "Hello!", result)
}

func TestLLMComplete_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	_, err := c.Complete(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestLLMComplete_NoChoices(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	_, err := c.Complete(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestLLMComplete_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	_, err := c.Complete(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestLLMComplete_RateLimitRetry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`rate limited`))
			return
		}
		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "ok"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	result, err := c.Complete(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.GreaterOrEqual(t, int(calls.Load()), 3)
}

func TestLLMComplete_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Complete(ctx, "test")
	require.Error(t, err)
}

func TestLLMComplete_RateLimitExhausted(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`rate limited`))
	}))
	defer srv.Close()

	c := NewLLMClient("key", "m")
	c.baseURL = srv.URL

	// Cancel context after a short time so retries are cut short
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := c.Complete(ctx, "test")
	require.Error(t, err)
	// Should have attempted at least once and then been cancelled or rate-limited
	assert.GreaterOrEqual(t, int(calls.Load()), 1)
}
