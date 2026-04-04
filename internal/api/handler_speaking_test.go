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

func TestSpeakingHandler_Evaluate_InvalidBody(t *testing.T) {
	t.Parallel()
	h := NewSpeakingHandler(&mockTranscriber{})

	req := httptest.NewRequest(http.MethodPost, "/api/speaking/evaluate", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.Evaluate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "invalid request body")
}

// --- buildFeedback tests ---

func TestBuildFeedback_AllScoreRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		score    float64
		contains string
	}{
		{"excellent_perfect", 1.0, "Excellent"},
		{"excellent_boundary", 0.95, "Excellent"},
		{"very_good_high", 0.90, "Very good"},
		{"very_good_boundary", 0.80, "Very good"},
		{"good_effort_mid", 0.70, "Good effort"},
		{"good_effort_boundary", 0.60, "Good effort"},
		{"keep_practicing_mid", 0.50, "Keep practicing"},
		{"keep_practicing_boundary", 0.40, "Keep practicing"},
		{"try_again_low", 0.30, "try again"},
		{"try_again_zero", 0.0, "try again"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			feedback := buildFeedback(tc.score, "transcript", "expected")
			assert.Contains(t, feedback, tc.contains,
				"score=%.2f should contain %q, got %q", tc.score, tc.contains, feedback)
		})
	}
}

// --- scoreSimilarity edge cases ---

func TestScoreSimilarity_BothEmpty(t *testing.T) {
	t.Parallel()
	// Both empty after normalization should return 1.0 (identical)
	score := scoreSimilarity("", "")
	assert.InDelta(t, 1.0, score, 0.001)
}

func TestScoreSimilarity_OneEmpty(t *testing.T) {
	t.Parallel()
	score := scoreSimilarity("hello", "")
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestScoreSimilarity_IdenticalWithPunctuation(t *testing.T) {
	t.Parallel()
	score := scoreSimilarity("Hello, World!", "hello world")
	assert.InDelta(t, 1.0, score, 0.001)
}

// --- levenshtein edge cases ---

func TestLevenshtein_BothEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, levenshtein("", ""))
}

func TestLevenshtein_OneEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 5, levenshtein("hello", ""))
	assert.Equal(t, 5, levenshtein("", "hello"))
}

func TestLevenshtein_Identical(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, levenshtein("hello", "hello"))
}

func TestLevenshtein_SingleEdit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, levenshtein("hello", "hallo"))
}
