package generator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

// TTSClient generates speech audio from text via an OpenRouter-compatible
// chat completions API with audio output modality.
type TTSClient struct {
	apiKey  string
	model   string
	voice   string
	format  string
	baseURL string
	client  *http.Client
	sem     chan struct{} // concurrency limiter
}

// NewTTSClient creates a TTS client. Uses OpenRouter's chat completions
// streaming API with audio output modality.
// If model is empty, defaults to "openai/gpt-audio-mini".
// Concurrency is capped at 3.
func NewTTSClient(apiKey, model, voice, baseURL string) *TTSClient {
	if model == "" {
		model = "openai/gpt-audio-mini"
	}
	if voice == "" {
		voice = "alloy"
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return &TTSClient{
		apiKey:  apiKey,
		model:   model,
		voice:   voice,
		format:  "pcm16",
		baseURL: baseURL,
		client:  &http.Client{},
		sem:     make(chan struct{}, 3),
	}
}

type ttsChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ttsAudioConfig struct {
	Voice  string `json:"voice"`
	Format string `json:"format"`
}

type ttsChatRequest struct {
	Model      string           `json:"model"`
	Messages   []ttsChatMessage `json:"messages"`
	Modalities []string         `json:"modalities"`
	Audio      ttsAudioConfig   `json:"audio"`
	Stream     bool             `json:"stream"`
}

type streamDelta struct {
	Audio *struct {
		Data       string `json:"data"`
		Transcript string `json:"transcript"`
	} `json:"audio,omitempty"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

// Synthesize converts text to audio bytes via streaming chat completions.
// It blocks until a concurrency slot is available (max 3 concurrent requests).
func (c *TTSClient) Synthesize(ctx context.Context, text string) ([]byte, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	prompt := fmt.Sprintf("Please read the following text aloud clearly and naturally, as if teaching a language student. Speak at a slightly slow pace:\n\n%s", text)

	body, err := json.Marshal(ttsChatRequest{
		Model: c.model,
		Messages: []ttsChatMessage{
			{Role: "user", Content: prompt},
		},
		Modalities: []string{"text", "audio"},
		Audio:      ttsAudioConfig{Voice: c.voice, Format: c.format},
		Stream:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tts: status %d: %s", resp.StatusCode, string(respBody))
	}

	return c.readSSEAudio(resp.Body)
}

// readSSEAudio reads SSE stream and collects base64 audio chunks.
func (c *TTSClient) readSSEAudio(r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 256*1024), 10*1024*1024)

	var audioChunks []string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Audio != nil && choice.Delta.Audio.Data != "" {
				audioChunks = append(audioChunks, choice.Delta.Audio.Data)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("tts: read stream: %w", err)
	}

	if len(audioChunks) == 0 {
		return nil, fmt.Errorf("tts: no audio data in response")
	}

	fullB64 := strings.Join(audioChunks, "")
	pcmBytes, err := base64.StdEncoding.DecodeString(fullB64)
	if err != nil {
		return nil, fmt.Errorf("tts: decode audio base64: %w", err)
	}

	return wrapPCM16InWAV(pcmBytes, 24000), nil
}

// SynthesizeBatch generates audio for multiple texts concurrently (up to 3 at a time).
// Returns a map from index to audio bytes. Errors are logged but don't fail the batch.
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

// wrapPCM16InWAV wraps raw 16-bit PCM samples in a WAV container.
// sampleRate is typically 24000 for OpenAI audio models. Mono, 16-bit.
func wrapPCM16InWAV(pcm []byte, sampleRate uint32) []byte {
	const (
		numChannels   = 1
		bitsPerSample = 16
	)
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := uint16(numChannels * bitsPerSample / 8)
	dataSize := uint32(len(pcm))

	var buf bytes.Buffer
	buf.Grow(44 + len(pcm))

	// RIFF header
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	// fmt sub-chunk
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16)) // sub-chunk size
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM format
	_ = binary.Write(&buf, binary.LittleEndian, uint16(numChannels))
	_ = binary.Write(&buf, binary.LittleEndian, sampleRate)
	_ = binary.Write(&buf, binary.LittleEndian, byteRate)
	_ = binary.Write(&buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	// data sub-chunk
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(pcm)

	return buf.Bytes()
}
