package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WhisperClient transcribes audio via OpenRouter's chat completions API
// using a model with audio input modality.
type WhisperClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewWhisperClient creates a transcription client using OpenRouter chat completions
// with audio input modality.
func NewWhisperClient(apiKey, model, baseURL string) *WhisperClient {
	if model == "" {
		model = "google/gemini-2.5-flash"
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return &WhisperClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type whisperChatRequest struct {
	Model    string           `json:"model"`
	Messages []whisperChatMsg `json:"messages"`
}

type whisperChatMsg struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type whisperTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type whisperAudioContent struct {
	Type       string            `json:"type"`
	InputAudio whisperInputAudio `json:"input_audio"`
}

type whisperInputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type whisperChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Transcribe sends audio bytes to the OpenRouter chat completions API and returns the transcription.
func (c *WhisperClient) Transcribe(ctx context.Context, audio []byte) (string, error) {
	b64Audio := base64.StdEncoding.EncodeToString(audio)

	reqBody := whisperChatRequest{
		Model: c.model,
		Messages: []whisperChatMsg{
			{
				Role: "user",
				Content: []interface{}{
					whisperTextContent{Type: "text", Text: "Transcribe the following audio exactly. Return only the transcribed text, nothing else."},
					whisperAudioContent{
						Type: "input_audio",
						InputAudio: whisperInputAudio{
							Data:   b64Audio,
							Format: "webm",
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("whisper: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("whisper: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
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

	var chatResp whisperChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("whisper: parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("whisper: no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
