package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// TTSClient generates speech audio from text via an OpenAI-compatible TTS API.
type TTSClient struct {
	apiKey  string
	model   string
	voice   string
	baseURL string
	client  *http.Client
	sem     chan struct{} // concurrency limiter
}

// NewTTSClient creates a TTS client. baseURL should be an OpenAI-compatible
// endpoint (e.g. "https://api.openai.com/v1" or a self-hosted alternative).
// If model is empty, defaults to "tts-1". Concurrency is capped at 3.
func NewTTSClient(apiKey, model, voice, baseURL string) *TTSClient {
	if model == "" {
		model = "tts-1"
	}
	if voice == "" {
		voice = "alloy"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &TTSClient{
		apiKey:  apiKey,
		model:   model,
		voice:   voice,
		baseURL: baseURL,
		client:  &http.Client{},
		sem:     make(chan struct{}, 3),
	}
}

type ttsRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

// Synthesize converts text to MP3 audio. It blocks until a concurrency slot
// is available (max 3 concurrent requests).
func (c *TTSClient) Synthesize(ctx context.Context, text string) ([]byte, error) {
	// Acquire semaphore
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	body, err := json.Marshal(ttsRequest{
		Model:          c.model,
		Input:          text,
		Voice:          c.voice,
		ResponseFormat: "mp3",
		Speed:          0.9, // slightly slower for language learning
	})
	if err != nil {
		return nil, fmt.Errorf("tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tts: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts: status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// SynthesizeBatch generates audio for multiple texts concurrently (up to 3 at a time).
// Returns a map from index to MP3 bytes. Errors are logged but don't fail the batch.
func (c *TTSClient) SynthesizeBatch(ctx context.Context, texts []string) map[int][]byte {
	results := make(map[int][]byte)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			data, err := c.Synthesize(ctx, t)
			if err != nil {
				slog.Warn("tts synthesis failed", "index", idx, "err", err)
				return
			}
			mu.Lock()
			results[idx] = data
			mu.Unlock()
		}(i, text)
	}

	wg.Wait()
	return results
}
