package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Transcriber converts audio bytes to text.
type Transcriber interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

// SpeakingHandler handles speaking evaluation requests.
type SpeakingHandler struct {
	transcriber Transcriber
}

// NewSpeakingHandler creates a SpeakingHandler.
func NewSpeakingHandler(t Transcriber) *SpeakingHandler {
	return &SpeakingHandler{transcriber: t}
}

type evaluateRequest struct {
	AudioBase64  string `json:"audio_base64"`
	ExpectedText string `json:"expected_text"`
}

type evaluateResponse struct {
	Transcript string  `json:"transcript"`
	Expected   string  `json:"expected"`
	Score      float64 `json:"score"`
	Feedback   string  `json:"feedback"`
}

// Evaluate handles POST /api/speaking/evaluate.
// Accepts base64-encoded audio, transcribes it, and scores against expected text.
func (h *SpeakingHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AudioBase64 == "" || req.ExpectedText == "" {
		writeError(w, http.StatusBadRequest, "audio_base64 and expected_text are required")
		return
	}

	audio, err := base64.StdEncoding.DecodeString(req.AudioBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid base64 audio data")
		return
	}

	transcript, err := h.transcriber.Transcribe(r.Context(), audio)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "transcription failed")
		return
	}

	score := scoreSimilarity(transcript, req.ExpectedText)
	feedback := buildFeedback(score, transcript, req.ExpectedText)

	writeJSON(w, http.StatusOK, evaluateResponse{
		Transcript: transcript,
		Expected:   req.ExpectedText,
		Score:      score,
		Feedback:   feedback,
	})
}

// scoreSimilarity computes a 0.0-1.0 similarity score between transcript and expected text.
// Uses normalized Levenshtein distance on lowercased, NFD-normalized strings.
func scoreSimilarity(transcript, expected string) float64 {
	a := normalize(transcript)
	b := normalize(expected)

	if a == b {
		return 1.0
	}

	dist := levenshtein(a, b)
	maxLen := len([]rune(a))
	if l := len([]rune(b)); l > maxLen {
		maxLen = l
	}
	if maxLen == 0 {
		return 0.0
	}

	return 1.0 - float64(dist)/float64(maxLen)
}

// normalize strips punctuation, lowercases, and applies NFD normalization.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = norm.NFD.String(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func buildFeedback(score float64, transcript, expected string) string {
	switch {
	case score >= 0.95:
		return "Excellent! Perfect pronunciation."
	case score >= 0.8:
		return "Very good! Minor differences detected."
	case score >= 0.6:
		return "Good effort. Some words need practice."
	case score >= 0.4:
		return "Keep practicing. Try listening to the correct pronunciation again."
	default:
		return "Let's try again. Listen carefully to each word."
	}
}
