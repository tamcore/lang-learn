package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
)

func setupAdminTest(t *testing.T) (*AdminHandler, *store.FileUserStore, *store.FileCourseStore) {
	t.Helper()
	dir := testutil.TempDataDir(t)

	users, err := store.NewFileUserStore(dir + "/users")
	require.NoError(t, err)

	courses, err := store.NewFileCourseStore(dir + "/courses")
	require.NoError(t, err)

	audit, err := store.NewFileAuditStore(dir + "/audit")
	require.NoError(t, err)

	h := NewAdminHandler(users, courses, audit, 4)
	return h, users, courses
}

func TestAdminListUsers(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "phil", Email: "phil@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	rec := httptest.NewRecorder()
	h.ListUsers(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.([]any)
	assert.Len(t, data, 1)
}

func TestAdminGetUser(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "phil", Email: "phil@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Get("/api/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users/u1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminGetUser_NotFound(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	r := chi.NewRouter()
	r.Get("/api/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users/nope", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminUpdateUser(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "old", Email: "old@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"username":"new","is_admin":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "new", data["username"])
	assert.True(t, data["is_admin"].(bool))
}

func TestAdminDeleteUser(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "del", Email: "del@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminListCourses(t *testing.T) {
	t.Parallel()
	h, _, cs := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, cs.Create(context.Background(), models.Course{
		ID: "c1", Title: "Test", SourceLang: "en", TargetLang: "sk",
		Direction: models.DirectionForward, Perspective: models.PerspectiveMale,
		CreatedAt: now, GeneratedAt: now, GeneratedBy: "test", Lessons: []models.Lesson{},
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/courses", nil)
	rec := httptest.NewRecorder()
	h.ListCourses(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminDeleteCourse(t *testing.T) {
	t.Parallel()
	h, _, cs := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, cs.Create(context.Background(), models.Course{
		ID: "c1", Title: "Test", SourceLang: "en", TargetLang: "sk",
		Direction: models.DirectionForward, Perspective: models.PerspectiveMale,
		CreatedAt: now, GeneratedAt: now, GeneratedBy: "test", Lessons: []models.Lesson{},
	}))

	r := chi.NewRouter()
	r.Delete("/api/admin/courses/{id}", h.DeleteCourse)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/courses/c1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminGetAudit(t *testing.T) {
	t.Parallel()
	dir := testutil.TempDataDir(t)

	users, err := store.NewFileUserStore(dir + "/users")
	require.NoError(t, err)
	courses, err := store.NewFileCourseStore(dir + "/courses")
	require.NoError(t, err)
	auditStore, err := store.NewFileAuditStore(dir + "/audit")
	require.NoError(t, err)

	h := NewAdminHandler(users, courses, auditStore, 4)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	rec := httptest.NewRecorder()
	h.GetAudit(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
