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
	assert.Equal(t, "whisper-1", c.model)
	assert.Equal(t, "https://api.openai.com/v1", c.baseURL)
}

func TestNewWhisperClient_Custom(t *testing.T) {
	t.Parallel()
	c := NewWhisperClient("key", "whisper-large-v3", "https://custom.api")
	assert.Equal(t, "whisper-large-v3", c.model)
	assert.Equal(t, "https://custom.api", c.baseURL)
}

func TestWhisperTranscribe_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/audio/transcriptions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		require.NoError(t, r.ParseMultipartForm(10<<20))
		assert.Equal(t, "whisper-1", r.FormValue("model"))

		_, fh, err := r.FormFile("file")
		require.NoError(t, err)
		assert.Equal(t, "audio.webm", fh.Filename)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(whisperResponse{Text: "Hello world"})
	}))
	defer srv.Close()

	c := NewWhisperClient("test-key", "", srv.URL)
	text, err := c.Transcribe(context.Background(), []byte("fake-audio-data"))
	require.NoError(t, err)
	assert.Equal(t, "Hello world", text)
}

func TestWhisperTranscribe_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestWhisperTranscribe_EmptyAudio(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(10<<20))
		_, fh, err := r.FormFile("file")
		require.NoError(t, err)
		assert.Equal(t, "audio.webm", fh.Filename)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(whisperResponse{Text: ""})
	}))
	defer srv.Close()

	c := NewWhisperClient("key", "", srv.URL)
	text, err := c.Transcribe(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Equal(t, "", text)
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

func TestWhisperTranscribe_LargeAudio(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(10<<20))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(whisperResponse{Text: "transcribed"})
	}))
	defer srv.Close()

	c := NewWhisperClient("key", "", srv.URL)
	largeAudio := make([]byte, 1024*1024) // 1MB
	text, err := c.Transcribe(context.Background(), largeAudio)
	require.NoError(t, err)
	assert.Equal(t, "transcribed", text)
}
