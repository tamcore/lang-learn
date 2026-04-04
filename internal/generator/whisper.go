package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// WhisperClient transcribes audio via an OpenAI-compatible Whisper API.
type WhisperClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewWhisperClient creates a Whisper transcription client.
func NewWhisperClient(apiKey, model, baseURL string) *WhisperClient {
	if model == "" {
		model = "whisper-1"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &WhisperClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type whisperResponse struct {
	Text string `json:"text"`
}

// Transcribe sends audio bytes to the Whisper API and returns the transcription.
func (c *WhisperClient) Transcribe(ctx context.Context, audio []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("model", c.model); err != nil {
		return "", fmt.Errorf("whisper: write model field: %w", err)
	}

	part, err := w.CreateFormFile("file", "audio.webm")
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("whisper: write audio data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("whisper: close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", fmt.Errorf("whisper: create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper: send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("whisper: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper: status %d: %s", resp.StatusCode, string(respBody))
	}

	var whisperResp whisperResponse
	if err := json.Unmarshal(respBody, &whisperResp); err != nil {
		return "", fmt.Errorf("whisper: parse response: %w", err)
	}

	return whisperResp.Text, nil
}
