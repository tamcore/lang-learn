package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LLMClient calls the OpenRouter chat completions API.
type LLMClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewLLMClient creates an LLM client.
func NewLLMClient(apiKey, model string) *LLMClient {
	if model == "" {
		model = "google/gemini-2.5-flash-preview"
	}
	return &LLMClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://openrouter.ai/api/v1",
		client:  &http.Client{},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete sends a prompt to the LLM and returns the response content.
func (c *LLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llm: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("llm: parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("llm: no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
