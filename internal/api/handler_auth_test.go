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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret-key-at-least-32-chars!!"

func newTestAuthHandler(t *testing.T) (*AuthHandler, *store.FileUserStore) {
	t.Helper()
	dir := testutil.TempDataDir(t)
	users, err := store.NewFileUserStore(dir + "/users")
	require.NoError(t, err)
	h := NewAuthHandler(users, testSecret, 15*time.Minute, 7*24*time.Hour, 4) // low bcrypt cost for speed
	return h, users
}

func TestLogin_MissingFields(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_UnknownUser(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"nobody","password":"password1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	h, users := newTestAuthHandler(t)

	// Create user directly in store
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1234"), 4)
	now := time.Now().UTC()
	require.NoError(t, users.Create(context.Background(), models.User{
		ID: "u1", Username: "testuser", PasswordHash: string(hash),
		CreatedAt: now, UpdatedAt: now,
	}))

	body := `{"username":"testuser","password":"pass1234","remember_me":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
}

func TestLogin_NoRememberMe(t *testing.T) {
	t.Parallel()
	h, users := newTestAuthHandler(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1234"), 4)
	now := time.Now().UTC()
	require.NoError(t, users.Create(context.Background(), models.User{
		ID: "u1", Username: "testuser", PasswordHash: string(hash),
		CreatedAt: now, UpdatedAt: now,
	}))

	body := `{"username":"testuser","password":"pass1234"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["access_token"])
	// No refresh token when remember_me is false
	_, hasRefresh := data["refresh_token"]
	assert.False(t, hasRefresh)
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	h, users := newTestAuthHandler(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct1"), 4)
	now := time.Now().UTC()
	require.NoError(t, users.Create(context.Background(), models.User{
		ID: "u1", Username: "testuser", PasswordHash: string(hash),
		CreatedAt: now, UpdatedAt: now,
	}))

	body := `{"username":"testuser","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefresh_Success(t *testing.T) {
	t.Parallel()
	h, users := newTestAuthHandler(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password1"), 4)
	now := time.Now().UTC()
	require.NoError(t, users.Create(context.Background(), models.User{
		ID: "u1", Username: "refreshuser", PasswordHash: string(hash),
		CreatedAt: now, UpdatedAt: now,
	}))

	// Login with remember_me to get refresh token
	body := `{"username":"refreshuser","password":"password1","remember_me":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	refreshToken := data["refresh_token"].(string)

	// Refresh
	body = `{"refresh_token":"` + refreshToken + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec = httptest.NewRecorder()
	h.Refresh(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRefresh_InvalidToken(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"refresh_token":"bad.token.here"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Login error path tests ---

func TestLogin_InvalidBody(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "invalid request body")
}

func TestLogin_StoreGetByUsernameError(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(
		&mockUserStore{getByUsernameFn: func(_ context.Context, _ string) (models.User, error) {
			return models.User{}, errors.New("db down")
		}},
		testSecret, 15*time.Minute, 7*24*time.Hour, 4,
	)

	body := `{"username":"someone","password":"password1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Refresh error path tests ---

func TestRefresh_InvalidBody(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefresh_EmptyToken(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"refresh_token":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "refresh_token is required")
}

func TestRefresh_ExpiredToken(t *testing.T) {
	t.Parallel()
	h, users := newTestAuthHandler(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password1"), 4)
	now := time.Now().UTC()
	require.NoError(t, users.Create(context.Background(), models.User{
		ID: "u1", Username: "expuser", PasswordHash: string(hash),
		CreatedAt: now, UpdatedAt: now,
	}))

	// Issue a token that's already expired (negative TTL)
	expiredToken, err := auth.IssueToken(testSecret, "u1", false, -1*time.Hour)
	require.NoError(t, err)

	body := `{"refresh_token":"` + expiredToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefresh_UserNotFound(t *testing.T) {
	t.Parallel()
	// Token is valid but user ID doesn't exist in store
	h := NewAuthHandler(
		&mockUserStore{getByIDFn: func(_ context.Context, _ string) (models.User, error) {
			return models.User{}, store.ErrNotFound
		}},
		testSecret, 15*time.Minute, 7*24*time.Hour, 4,
	)

	validToken, err := auth.IssueToken(testSecret, "deleted-user", false, 15*time.Minute)
	require.NoError(t, err)

	body := `{"refresh_token":"` + validToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Contains(t, env.Error, "user not found")
}
