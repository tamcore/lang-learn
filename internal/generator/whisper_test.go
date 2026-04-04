package generator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWhisperClient_Defaults(t *testing.T) {
	t.Parallel()
	c := NewWhisperClient("key", "", "")
	assert.Equal(t, "google/gemini-2.5-flash", c.model)
	assert.Equal(t, "https://openrouter.ai/api/v1", c.baseURL)
}

func TestNewWhisperClient_Custom(t *testing.T) {
	t.Parallel()
	c := NewWhisperClient("key", "google/gemini-2.0-flash-001", "https://custom.api")
	assert.Equal(t, "google/gemini-2.0-flash-001", c.model)
	assert.Equal(t, "https://custom.api", c.baseURL)
}

func TestWhisperTranscribe_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body has audio content
		var reqBody whisperChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Len(t, reqBody.Messages, 1)
		assert.Equal(t, "user", reqBody.Messages[0].Role)

		w.Header().Set("Content-Type", "application/json")
		resp := whisperChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "Hello world"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewWhisperClient("test-key", "", srv.URL)
	text, err := c.Transcribe(context.Background(), []byte("fake-audio-data"))
	require.NoError(t, err)
	assert.Equal(t, "Hello world", text)
}

func TestWhisperTranscribe_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewWhisperClient("key", "", srv.URL)
	_, err := c.Transcribe(context.Background(), []byte("audio"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestWhisperTranscribe_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewWhisperClient("key", "", srv.URL)
	_, err := c.Transcribe(ctx, []byte("audio"))
	assert.Error(t, err)
}

func TestWhisperTranscribe_NoChoices(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(whisperChatResponse{})
	}))
	defer srv.Close()

	c := NewWhisperClient("key", "", srv.URL)
	_, err := c.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestWhisperTranscribe_InvalidJSONResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewWhisperClient("key", "", srv.URL)
	_, err := c.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}
