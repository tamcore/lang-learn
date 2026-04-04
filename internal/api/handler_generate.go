package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/generator"
	"github.com/user/lang-learn/internal/models"
)

// GenerateHandler manages course generation jobs.
type GenerateHandler struct {
	gen *generator.Generator
}

// NewGenerateHandler creates a GenerateHandler.
func NewGenerateHandler(gen *generator.Generator) *GenerateHandler {
	return &GenerateHandler{gen: gen}
}

// Generate handles POST /api/admin/courses/generate.
func (h *GenerateHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BlueprintID string `json:"blueprint_id"`
		SourceLang  string `json:"source_lang"`
		TargetLang  string `json:"target_lang"`
		Direction   string `json:"direction"`
		Perspective string `json:"perspective"`
		LessonCount int    `json:"lesson_count"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.BlueprintID == "" || req.SourceLang == "" || req.TargetLang == "" {
		writeError(w, http.StatusBadRequest, "blueprint_id, source_lang, target_lang are required")
		return
	}
	if req.LessonCount <= 0 {
		req.LessonCount = 5
	}
	if req.Direction == "" {
		req.Direction = "forward"
	}
	if req.Perspective == "" {
		req.Perspective = "male"
	}

	actorID := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorID = claims.UserID
	}

	jobID := h.gen.Generate(generator.GenerateRequest{
		BlueprintID: req.BlueprintID,
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Direction:   models.CourseDirection(req.Direction),
		Perspective: models.Perspective(req.Perspective),
		LessonCount: req.LessonCount,
		ActorID:     actorID,
	})

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// GetJobStatus handles GET /api/admin/courses/generate/{jobID}.
func (h *GenerateHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	job, ok := h.gen.GetJob(jobID)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}
