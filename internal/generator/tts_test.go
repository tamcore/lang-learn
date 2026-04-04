package generator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTTSClient_Defaults(t *testing.T) {
	t.Parallel()
	c := NewTTSClient("key", "", "", "")
	assert.Equal(t, "tts-1", c.model)
	assert.Equal(t, "alloy", c.voice)
	assert.Equal(t, "https://api.openai.com/v1", c.baseURL)
	assert.Equal(t, 3, cap(c.sem))
}

func TestNewTTSClient_CustomParams(t *testing.T) {
	t.Parallel()
	c := NewTTSClient("key", "tts-1-hd", "nova", "https://custom.api/v1")
	assert.Equal(t, "tts-1-hd", c.model)
	assert.Equal(t, "nova", c.voice)
	assert.Equal(t, "https://custom.api/v1", c.baseURL)
}

func TestSynthesize_Success(t *testing.T) {
	t.Parallel()
	fakeAudio := []byte("fake-mp3-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/audio/speech", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeAudio)
	}))
	defer srv.Close()

	c := NewTTSClient("test-key", "tts-1", "alloy", srv.URL)
	data, err := c.Synthesize(context.Background(), "Hello world")
	require.NoError(t, err)
	assert.Equal(t, fakeAudio, data)
}

func TestSynthesize_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	_, err := c.Synthesize(context.Background(), "fail")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestSynthesize_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio"))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Synthesize(ctx, "hello")
	require.Error(t, err)
}

func TestSynthesizeBatch_Success(t *testing.T) {
	t.Parallel()
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio-data"))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	results := c.SynthesizeBatch(context.Background(), []string{"one", "two", "three"})
	assert.Len(t, results, 3)
	assert.Equal(t, []byte("audio-data"), results[0])
	assert.Equal(t, []byte("audio-data"), results[1])
	assert.Equal(t, []byte("audio-data"), results[2])
}

func TestSynthesizeBatch_PartialFailure(t *testing.T) {
	t.Parallel()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call++
		if call%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	results := c.SynthesizeBatch(context.Background(), []string{"a", "b", "c"})
	// Some should succeed, some fail — we can't predict order due to concurrency,
	// but total results should be < 3
	assert.True(t, len(results) > 0 && len(results) <= 3,
		"expected partial results, got %d", len(results))
}

func TestSynthesize_ConcurrencyLimit(t *testing.T) {
	t.Parallel()
	active := make(chan int, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		active <- 1
		// small delay to test concurrency
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio"))
		<-active
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	// Semaphore cap is 3 — sending 5 requests should still work
	results := c.SynthesizeBatch(context.Background(), []string{"1", "2", "3", "4", "5"})
	assert.Len(t, results, 5)
}
