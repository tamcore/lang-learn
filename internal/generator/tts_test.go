package generator

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTTSClient_Defaults(t *testing.T) {
	t.Parallel()
	c := NewTTSClient("key", "", "", "")
	assert.Equal(t, "openai/gpt-audio-mini", c.model)
	assert.Equal(t, "alloy", c.voice)
	assert.Equal(t, "https://openrouter.ai/api/v1", c.baseURL)
	assert.Equal(t, 3, cap(c.sem))
}

func TestNewTTSClient_CustomParams(t *testing.T) {
	t.Parallel()
	c := NewTTSClient("key", "openai/gpt-audio", "nova", "https://custom.api/v1")
	assert.Equal(t, "openai/gpt-audio", c.model)
	assert.Equal(t, "nova", c.voice)
	assert.Equal(t, "https://custom.api/v1", c.baseURL)
}

// sseAudioResponse builds a fake SSE stream with base64 audio chunks.
func sseAudioResponse(w http.ResponseWriter, audioData []byte) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	b64 := base64.StdEncoding.EncodeToString(audioData)
	mid := len(b64) / 2
	chunk1 := b64[:mid]
	chunk2 := b64[mid:]

	_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"audio\":{\"data\":\"%s\",\"transcript\":\"hello\"}}}]}\n\n", chunk1)
	_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"audio\":{\"data\":\"%s\",\"transcript\":\" world\"}}}]}\n\n", chunk2)
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
}

func TestSynthesize_Success(t *testing.T) {
	t.Parallel()
	fakeAudio := []byte("fake-audio-data-for-tts-test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		sseAudioResponse(w, fakeAudio)
	}))
	defer srv.Close()

	c := NewTTSClient("test-key", "openai/gpt-audio-mini", "alloy", srv.URL)
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

func TestSynthesize_NoAudioInResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	_, err := c.Synthesize(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no audio data")
}

func TestSynthesize_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseAudioResponse(w, []byte("audio"))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Synthesize(ctx, "hello")
	require.Error(t, err)
}

func TestSynthesizeBatch_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseAudioResponse(w, []byte("audio-data"))
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
		sseAudioResponse(w, []byte("ok"))
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	results := c.SynthesizeBatch(context.Background(), []string{"a", "b", "c"})
	assert.True(t, len(results) > 0 && len(results) <= 3,
		"expected partial results, got %d", len(results))
}

func TestSynthesize_ConcurrencyLimit(t *testing.T) {
	t.Parallel()
	active := make(chan int, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		active <- 1
		sseAudioResponse(w, []byte("audio"))
		<-active
	}))
	defer srv.Close()

	c := NewTTSClient("key", "", "", srv.URL)
	results := c.SynthesizeBatch(context.Background(), []string{"1", "2", "3", "4", "5"})
	assert.Len(t, results, 5)
}
