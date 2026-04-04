package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
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

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"philipp","email":"philipp@test.com","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	user := data["user"].(map[string]any)
	assert.Equal(t, "philipp", user["username"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"u1","email":"dup@test.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec2 := httptest.NewRecorder()
	h.Register(rec2, req2)
	assert.Equal(t, http.StatusConflict, rec2.Code)
}

func TestRegister_MissingFields(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"","email":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	// Register first
	body := `{"username":"login_test","email":"login@test.com","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Login
	body = `{"email":"login@test.com","password":"pass123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec = httptest.NewRecorder()
	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["access_token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"username":"u","email":"wrong@test.com","password":"correct"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	body = `{"email":"wrong@test.com","password":"incorrect"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_UnknownEmail(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	body := `{"email":"nobody@test.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefresh_Success(t *testing.T) {
	t.Parallel()
	h, _ := newTestAuthHandler(t)

	// Register to get tokens
	body := `{"username":"refresh","email":"refresh@test.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

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
