package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/generator"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
)

func setupGenerateTest(t *testing.T) (*GenerateHandler, *generator.Generator) {
	t.Helper()
	dir := testutil.TempDataDir(t)

	courses, err := store.NewFileCourseStore(dir + "/courses")
	require.NoError(t, err)
	audit, err := store.NewFileAuditStore(dir + "/audit")
	require.NoError(t, err)

	// nil LLM and TTS — use a non-existent blueprint so the background
	// goroutine fails gracefully before calling the nil LLM client.
	gen := generator.NewGenerator(nil, nil, courses, audit, dir)
	h := NewGenerateHandler(gen)
	return h, gen
}

func TestNewGenerateHandler(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)
	assert.NotNil(t, h)
}

func TestGenerate_Success(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	// Use a fake blueprint so the background goroutine exits early without
	// calling the nil LLM client.
	body := `{"blueprint_id":"fake-bp","source_lang":"en","target_lang":"sk","lesson_count":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["job_id"])
}

func TestGenerate_DefaultValues(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	// Omit direction, perspective, lesson_count — handler should fill defaults
	body := `{"blueprint_id":"fake-bp","source_lang":"en","target_lang":"sk"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestGenerate_MissingFields(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	body := `{"blueprint_id":"","source_lang":"","target_lang":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGenerate_InvalidBody(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/generate", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetJobStatus_NotFound(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	r := chi.NewRouter()
	r.Get("/api/admin/courses/generate/{jobID}", h.GetJobStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/courses/generate/nonexistent", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetJobStatus_Found(t *testing.T) {
	t.Parallel()
	h, _ := setupGenerateTest(t)

	// Start a generate job with a fake blueprint so the goroutine exits
	// gracefully. The job will end up "failed" but it will be in the map.
	body := `{"blueprint_id":"fake-bp","source_lang":"en","target_lang":"sk"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/courses/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Generate(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	jobID := data["job_id"].(string)

	// Give the background goroutine a moment to run
	time.Sleep(50 * time.Millisecond)

	r := chi.NewRouter()
	r.Get("/api/admin/courses/generate/{jobID}", h.GetJobStatus)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/courses/generate/"+jobID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env2 envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env2))
	jobData := env2.Data.(map[string]any)
	assert.Equal(t, jobID, jobData["id"])
}
