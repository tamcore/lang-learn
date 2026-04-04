package api

import (
	"context"
	"encoding/json"
	"errors"
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

// --- CreateUser tests ---

func TestAdminCreateUser_Success(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	body := `{"username":"newuser","password":"longpassword","is_admin":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "newuser", data["username"])
	assert.False(t, data["is_admin"].(bool))
}

func TestAdminCreateUser_Admin(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	body := `{"username":"adminuser","password":"longpassword","is_admin":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.True(t, data["is_admin"].(bool))
}

func TestAdminCreateUser_MissingFields(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	body := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreateUser_ShortPassword(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	body := `{"username":"short","password":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "at least 8 characters")
}

func TestAdminCreateUser_InvalidBody(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminCreateUser_DuplicateUsername(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "taken", Email: "taken@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	body := `{"username":"taken","password":"longpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// --- UpdateUser additional tests ---

func TestAdminUpdateUser_NotFound(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"username":"whatever"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/nonexistent", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminUpdateUser_InvalidBody(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "old", Email: "old@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader("bad json"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUpdateUser_EmailOnly(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "phil", Email: "old@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"email":"new@test.com"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "phil", data["username"])
}

// --- DeleteUser additional tests ---

func TestAdminDeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	h, _, _ := setupAdminTest(t)

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/nonexistent", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminDeleteUser_VerifyRemoved(t *testing.T) {
	t.Parallel()
	h, us, _ := setupAdminTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, us.Create(context.Background(), models.User{
		ID: "u1", Username: "del", Email: "del@test.com", PasswordHash: "x", CreatedAt: now, UpdatedAt: now,
	}))

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", h.DeleteUser)
	r.Get("/api/admin/users/{id}", h.GetUser)

	// Delete
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify gone
	req = httptest.NewRequest(http.MethodGet, "/api/admin/users/u1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Mock-based error path tests ---

func TestAdminListUsers_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{listFn: func(_ context.Context) ([]models.User, error) {
			return nil, errors.New("db down")
		}},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	rec := httptest.NewRecorder()
	h.ListUsers(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "failed to list users")
}

func TestAdminGetUser_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{getByIDFn: func(_ context.Context, _ string) (models.User, error) {
			return models.User{}, errors.New("disk error")
		}},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Get("/api/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users/u1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminUpdateUser_StoreGetError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{getByIDFn: func(_ context.Context, _ string) (models.User, error) {
			return models.User{}, errors.New("disk error")
		}},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"username":"new"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminUpdateUser_StoreUpdateConflict(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{
			getByIDFn: func(_ context.Context, _ string) (models.User, error) {
				return models.User{ID: "u1", Username: "old"}, nil
			},
			updateFn: func(_ context.Context, _ models.User) error {
				return store.ErrConflict
			},
		},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"email":"dup@test.com"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "email already taken")
}

func TestAdminUpdateUser_StoreUpdateError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{
			getByIDFn: func(_ context.Context, _ string) (models.User, error) {
				return models.User{ID: "u1", Username: "old"}, nil
			},
			updateFn: func(_ context.Context, _ models.User) error {
				return errors.New("write fail")
			},
		},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Patch("/api/admin/users/{id}", h.UpdateUser)

	body := `{"username":"new"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/u1", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminDeleteUser_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{deleteFn: func(_ context.Context, _ string) error {
			return errors.New("disk error")
		}},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminListCourses_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{},
		&mockCourseStore{listFn: func(_ context.Context) ([]models.Course, error) {
			return nil, errors.New("db down")
		}},
		&mockAuditStore{}, 4,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/courses", nil)
	rec := httptest.NewRecorder()
	h.ListCourses(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "failed to list courses")
}

func TestAdminDeleteCourse_NotFound(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{},
		&mockCourseStore{deleteFn: func(_ context.Context, _ string) error {
			return store.ErrNotFound
		}},
		&mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Delete("/api/admin/courses/{id}", h.DeleteCourse)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/courses/nope", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminDeleteCourse_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{},
		&mockCourseStore{deleteFn: func(_ context.Context, _ string) error {
			return errors.New("disk error")
		}},
		&mockAuditStore{}, 4,
	)

	r := chi.NewRouter()
	r.Delete("/api/admin/courses/{id}", h.DeleteCourse)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/courses/c1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminGetAudit_InvalidDate(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(&mockUserStore{}, &mockCourseStore{}, &mockAuditStore{}, 4)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit?date=not-a-date", nil)
	rec := httptest.NewRecorder()
	h.GetAudit(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "invalid date format")
}

func TestAdminGetAudit_StoreError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{}, &mockCourseStore{},
		&mockAuditStore{listByDateFn: func(_ context.Context, _ time.Time) ([]models.AuditEntry, error) {
			return nil, errors.New("read fail")
		}}, 4,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit?date=2024-01-01", nil)
	rec := httptest.NewRecorder()
	h.GetAudit(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAdminCreateUser_StoreCreateError(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(
		&mockUserStore{createFn: func(_ context.Context, _ models.User) error {
			return errors.New("disk full")
		}},
		&mockCourseStore{}, &mockAuditStore{}, 4,
	)

	body := `{"username":"newuser","password":"longpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
