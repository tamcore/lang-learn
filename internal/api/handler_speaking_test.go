package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTranscriber implements the Transcriber interface for tests.
type mockTranscriber struct {
	transcript string
	err        error
}

func (m *mockTranscriber) Transcribe(_ context.Context, _ []byte) (string, error) {
	return m.transcript, m.err
}

func TestSpeakingHandler_Evaluate_Success(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{transcript: "Dobrý den"})

	body := `{"audio_base64":"SGVsbG8=","expected_text":"Dobrý den"}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "Dobrý den", data["transcript"])
	score := data["score"].(float64)
	assert.InDelta(t, 1.0, score, 0.01, "exact match should be ~1.0")
	assert.NotEmpty(t, data["feedback"])
}

func TestSpeakingHandler_Evaluate_PartialMatch(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{transcript: "Dobry denn please"})

	body := `{"audio_base64":"SGVsbG8=","expected_text":"Dobrý den"}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	score := data["score"].(float64)
	assert.True(t, score > 0.3 && score < 1.0, "partial match score should be 0.3-1.0, got %f", score)
}

func TestSpeakingHandler_Evaluate_NoMatch(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{transcript: "something completely different"})

	body := `{"audio_base64":"SGVsbG8=","expected_text":"Dobrý den"}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	score := data["score"].(float64)
	assert.True(t, score < 0.5, "no match should score < 0.5, got %f", score)
}

func TestSpeakingHandler_Evaluate_MissingFields(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{})

	body := `{"audio_base64":"","expected_text":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSpeakingHandler_Evaluate_InvalidBase64(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{})

	body := `{"audio_base64":"!!!not-base64!!!","expected_text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSpeakingHandler_Evaluate_TranscriberError(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{err: assert.AnError})

	body := `{"audio_base64":"SGVsbG8=","expected_text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
