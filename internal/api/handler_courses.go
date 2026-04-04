package api

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/store"
)

// CourseHandler groups course-related HTTP handlers.
type CourseHandler struct {
	courses  store.CourseStorer
	progress store.ProgressStorer
}

// NewCourseHandler creates a CourseHandler with the given dependencies.
func NewCourseHandler(courses store.CourseStorer, progress store.ProgressStorer) *CourseHandler {
	return &CourseHandler{courses: courses, progress: progress}
}

// ListCourses handles GET /api/courses — returns all courses without lesson details.
func (h *CourseHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.courses.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list courses")
		return
	}

	type courseSummary struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		SourceLang  string `json:"source_lang"`
		TargetLang  string `json:"target_lang"`
		Direction   string `json:"direction"`
		Perspective string `json:"perspective"`
		LessonCount int    `json:"lesson_count"`
	}

	summaries := make([]courseSummary, 0, len(courses))
	for _, c := range courses {
		summaries = append(summaries, courseSummary{
			ID:          c.ID,
			Title:       c.Title,
			Description: c.Description,
			SourceLang:  c.SourceLang,
			TargetLang:  c.TargetLang,
			Direction:   string(c.Direction),
			Perspective: string(c.Perspective),
			LessonCount: c.LessonCount,
		})
	}
	writeJSON(w, http.StatusOK, summaries)
}

// GetCourse handles GET /api/courses/{id} — returns course metadata + lesson list.
func (h *CourseHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	course, err := h.courses.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "course not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, course)
}

// GetLesson handles GET /api/courses/{id}/lessons/{seq} — returns full lesson with turns.
func (h *CourseHandler) GetLesson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	seqStr := chi.URLParam(r, "seq")

	course, err := h.courses.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "course not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var seq int
	for _, c := range seqStr {
		seq = seq*10 + int(c-'0')
	}

	for _, lesson := range course.Lessons {
		if lesson.Sequence == seq {
			writeJSON(w, http.StatusOK, lesson)
			return
		}
	}
	writeError(w, http.StatusNotFound, "lesson not found")
}

// GetProgress handles GET /api/progress — returns all progress for the current user.
func (h *CourseHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	progress, err := h.progress.ListByUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

// GetCourseProgress handles GET /api/progress/{courseID}.
func (h *CourseHandler) GetCourseProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	courseID := chi.URLParam(r, "courseID")
	p, err := h.progress.Get(r.Context(), claims.UserID, courseID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no progress found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// UpsertProgress handles PUT /api/progress/{courseID}.
func (h *CourseHandler) UpsertProgress(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	courseID := chi.URLParam(r, "courseID")

	var req struct {
		CurrentLesson int `json:"current_lesson"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.progress.Get(r.Context(), claims.UserID, courseID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	existing.UserID = claims.UserID
	existing.CourseID = courseID
	existing.CurrentLesson = req.CurrentLesson

	if err := h.progress.Upsert(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// AudioHandler serves course audio files.
type AudioHandler struct {
	audioBaseDir string
}

// NewAudioHandler creates an AudioHandler.
func NewAudioHandler(coursesDir string) *AudioHandler {
	return &AudioHandler{audioBaseDir: coursesDir}
}

// ServeAudio handles GET /api/audio/{courseID}/{filename} with Range header support.
func (h *AudioHandler) ServeAudio(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseID")
	filename := chi.URLParam(r, "filename")

	// Prevent path traversal
	if strings.Contains(courseID, "..") || strings.Contains(filename, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	path := filepath.Join(h.audioBaseDir, courseID, "audio", filename)
	http.ServeFile(w, r, path)
}
