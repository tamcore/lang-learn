package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
)

func setupCourseTest(t *testing.T) (*CourseHandler, *store.FileCourseStore, *store.FileProgressStore, string) {
	t.Helper()
	dir := testutil.TempDataDir(t)

	courses, err := store.NewFileCourseStore(dir + "/courses")
	require.NoError(t, err)

	progress, err := store.NewFileProgressStore(dir + "/progress")
	require.NoError(t, err)

	h := NewCourseHandler(courses, progress)
	return h, courses, progress, dir
}

func seedCourse(t *testing.T, cs *store.FileCourseStore, id string) models.Course {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	c := models.Course{
		ID:          id,
		Title:       "SK → EN",
		Description: "Slovak to English",
		SourceLang:  "sk",
		TargetLang:  "en",
		Direction:   models.DirectionForward,
		Perspective: models.PerspectiveMale,
		BlueprintID: "travel-basics-v1",
		LessonCount: 1,
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: "test",
		Lessons: []models.Lesson{
			{
				ID:       "l1",
				CourseID: id,
				Sequence: 1,
				Title:    "Lesson 1",
				Turns: []models.Turn{
					{ID: "t1", Sequence: 1, Speaker: models.SpeakerSystem, Text: "Hello", Translation: "Ahoj"},
				},
				CreatedAt: now,
			},
		},
	}
	require.NoError(t, cs.Create(context.Background(), c))
	return c
}

func TestListCourses(t *testing.T) {
	t.Parallel()
	h, cs, _, _ := setupCourseTest(t)

	seedCourse(t, cs, "c1")
	seedCourse(t, cs, "c2")

	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	rec := httptest.NewRecorder()
	h.ListCourses(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.([]any)
	assert.Len(t, data, 2)
}

func TestGetCourse(t *testing.T) {
	t.Parallel()
	h, cs, _, _ := setupCourseTest(t)
	seedCourse(t, cs, "c1")

	r := chi.NewRouter()
	r.Get("/api/courses/{id}", h.GetCourse)

	req := httptest.NewRequest(http.MethodGet, "/api/courses/c1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCourse_NotFound(t *testing.T) {
	t.Parallel()
	h, _, _, _ := setupCourseTest(t)

	r := chi.NewRouter()
	r.Get("/api/courses/{id}", h.GetCourse)

	req := httptest.NewRequest(http.MethodGet, "/api/courses/nope", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetLesson(t *testing.T) {
	t.Parallel()
	h, cs, _, _ := setupCourseTest(t)
	seedCourse(t, cs, "c1")

	r := chi.NewRouter()
	r.Get("/api/courses/{id}/lessons/{seq}", h.GetLesson)

	req := httptest.NewRequest(http.MethodGet, "/api/courses/c1/lessons/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetLesson_NotFound(t *testing.T) {
	t.Parallel()
	h, cs, _, _ := setupCourseTest(t)
	seedCourse(t, cs, "c1")

	r := chi.NewRouter()
	r.Get("/api/courses/{id}/lessons/{seq}", h.GetLesson)

	req := httptest.NewRequest(http.MethodGet, "/api/courses/c1/lessons/99", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetProgress(t *testing.T) {
	t.Parallel()
	h, _, ps, _ := setupCourseTest(t)

	require.NoError(t, ps.Upsert(context.Background(), models.CourseProgress{
		UserID: "user1", CourseID: "c1", CurrentLesson: 2,
	}))

	r := chi.NewRouter()
	r.Use(auth.RequireAuth(testSecret))
	r.Get("/api/progress", h.GetProgress)

	req := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	testutil.WithJWT(t, req, testSecret, "user1", false)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpsertProgress(t *testing.T) {
	t.Parallel()
	h, _, _, _ := setupCourseTest(t)

	r := chi.NewRouter()
	r.Use(auth.RequireAuth(testSecret))
	r.Put("/api/progress/{courseID}", h.UpsertProgress)

	body := `{"current_lesson":3}`
	req := httptest.NewRequest(http.MethodPut, "/api/progress/c1", strings.NewReader(body))
	testutil.WithJWT(t, req, testSecret, "user1", false)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetCourseProgress tests ---

func TestGetCourseProgress_Success(t *testing.T) {
	t.Parallel()
	h, _, ps, _ := setupCourseTest(t)

	require.NoError(t, ps.Upsert(context.Background(), models.CourseProgress{
		UserID: "user1", CourseID: "c1", CurrentLesson: 3,
	}))

	r := chi.NewRouter()
	r.Use(auth.RequireAuth(testSecret))
	r.Get("/api/progress/{courseID}", h.GetCourseProgress)

	req := httptest.NewRequest(http.MethodGet, "/api/progress/c1", nil)
	testutil.WithJWT(t, req, testSecret, "user1", false)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "c1", data["course_id"])
	assert.Equal(t, float64(3), data["current_lesson"])
}

func TestGetCourseProgress_NotFound(t *testing.T) {
	t.Parallel()
	h, _, _, _ := setupCourseTest(t)

	r := chi.NewRouter()
	r.Use(auth.RequireAuth(testSecret))
	r.Get("/api/progress/{courseID}", h.GetCourseProgress)

	req := httptest.NewRequest(http.MethodGet, "/api/progress/nonexistent", nil)
	testutil.WithJWT(t, req, testSecret, "user1", false)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCourseProgress_Unauthorized(t *testing.T) {
	t.Parallel()
	h, _, _, _ := setupCourseTest(t)

	r := chi.NewRouter()
	r.Use(auth.RequireAuth(testSecret))
	r.Get("/api/progress/{courseID}", h.GetCourseProgress)

	req := httptest.NewRequest(http.MethodGet, "/api/progress/c1", nil)
	// No JWT
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- AudioHandler tests ---

func TestNewAudioHandler(t *testing.T) {
	t.Parallel()
	h := NewAudioHandler("/some/path")
	assert.NotNil(t, h)
	assert.Equal(t, "/some/path", h.audioBaseDir)
}

func TestServeAudio_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	audioDir := dir + "/course1/audio"
	require.NoError(t, os.MkdirAll(audioDir, 0o755))
	require.NoError(t, os.WriteFile(audioDir+"/hello.mp3", []byte("fake-audio-data"), 0o644))

	h := NewAudioHandler(dir)

	r := chi.NewRouter()
	r.Get("/api/audio/{courseID}/{filename}", h.ServeAudio)

	req := httptest.NewRequest(http.MethodGet, "/api/audio/course1/hello.mp3", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "fake-audio-data", rec.Body.String())
}

func TestServeAudio_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	h := NewAudioHandler(dir)

	r := chi.NewRouter()
	r.Get("/api/audio/{courseID}/{filename}", h.ServeAudio)

	req := httptest.NewRequest(http.MethodGet, "/api/audio/course1/missing.mp3", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServeAudio_PathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	h := NewAudioHandler(dir)

	r := chi.NewRouter()
	r.Get("/api/audio/{courseID}/{filename}", h.ServeAudio)

	req := httptest.NewRequest(http.MethodGet, "/api/audio/..%2F../etc/passwd", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Should not return 200
	assert.NotEqual(t, http.StatusOK, rec.Code)
}
