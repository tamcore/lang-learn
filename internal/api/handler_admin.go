package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// AdminHandler groups admin HTTP handlers.
type AdminHandler struct {
	users      store.UserStorer
	courses    store.CourseStorer
	audit      store.AuditStorer
	bcryptCost int
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(users store.UserStorer, courses store.CourseStorer, audit store.AuditStorer, bcryptCost int) *AdminHandler {
	return &AdminHandler{users: users, courses: courses, audit: audit, bcryptCost: bcryptCost}
}

// ListUsers handles GET /api/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	dtos := make([]userDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, toUserDTO(u))
	}
	writeJSON(w, http.StatusOK, dtos)
}

// CreateUser handles POST /api/admin/users.
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:           generateID(),
		Username:     req.Username,
		PasswordHash: string(hash),
		IsAdmin:      req.IsAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, toUserDTO(user))
}

// GetUser handles GET /api/admin/users/{id}.
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}

// UpdateUser handles PATCH /api/admin/users/{id}.
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req struct {
		Username *string `json:"username"`
		Email    *string `json:"email"`
		IsAdmin  *bool   `json:"is_admin"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "email already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}

// DeleteUser handles DELETE /api/admin/users/{id}.
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.users.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

// ListCourses handles GET /api/admin/courses.
func (h *AdminHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.courses.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list courses")
		return
	}
	writeJSON(w, http.StatusOK, courses)
}

// DeleteCourse handles DELETE /api/admin/courses/{id}.
func (h *AdminHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.courses.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "course not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "course deleted"})
}

// GetAudit handles GET /api/admin/audit?date=YYYY-MM-DD.
func (h *AdminHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		return
	}

	entries, err := h.audit.ListByDate(r.Context(), date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query audit log")
		return
	}
	if entries == nil {
		entries = []models.AuditEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}
