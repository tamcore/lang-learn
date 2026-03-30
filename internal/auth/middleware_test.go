package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/auth"
)

// middlewareSecret is separate from testSecret in jwt_test.go to avoid redeclaration.
const middlewareSecret = "test-middleware-secret"

// okHandler responds 200 and writes the userID from context for inspection.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "no claims", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(claims.UserID))
})

func makeToken(t *testing.T, secret, userID string, isAdmin bool, ttl time.Duration) string {
	t.Helper()
	tok, err := auth.IssueToken(secret, userID, isAdmin, ttl)
	require.NoError(t, err)
	return tok
}

// --- RequireAuth tests ---

func TestRequireAuth_NoHeader(t *testing.T) {
	t.Parallel()
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_MalformedHeader_MissingBearer(t *testing.T) {
	t.Parallel()
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "not-bearer-format")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_MalformedHeader_BearerOnly(t *testing.T) {
	t.Parallel()
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	t.Parallel()
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.valid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_WrongSecret(t *testing.T) {
	t.Parallel()
	token := makeToken(t, "other-secret", "user-1", false, 15*time.Minute)
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "user-1", false, -1*time.Minute)
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_ValidToken(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "user-42", false, 15*time.Minute)
	handler := auth.RequireAuth(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-42", rec.Body.String())
}

func TestRequireAuth_TamperedToken(t *testing.T) {
	t.Parallel()
	// Build a token signed with a different key but claim admin=true.
	claims := jwt.MapClaims{
		"sub":      "attacker",
		"is_admin": true,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	handler := auth.RequireAuth(middlewareSecret)(okHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- RequireAdmin tests ---

func TestRequireAdmin_ValidAdmin(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "admin-1", true, 15*time.Minute)
	handler := auth.RequireAdmin(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin-1", rec.Body.String())
}

func TestRequireAdmin_NotAdmin(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "user-1", false, 15*time.Minute)
	handler := auth.RequireAdmin(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireAdmin_NoToken(t *testing.T) {
	t.Parallel()
	handler := auth.RequireAdmin(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAdmin_ExpiredToken(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "admin-1", true, -1*time.Minute)
	handler := auth.RequireAdmin(middlewareSecret)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- ClaimsFromContext tests ---

func TestClaimsFromContext_Empty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	claims, ok := auth.ClaimsFromContext(req.Context())
	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestClaimsFromContext_WithClaims(t *testing.T) {
	t.Parallel()
	token := makeToken(t, middlewareSecret, "user-99", true, 15*time.Minute)
	var captured *auth.Claims

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = auth.ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.RequireAuth(middlewareSecret)(sentinel)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NotNil(t, captured)
	assert.Equal(t, "user-99", captured.UserID)
	assert.True(t, captured.IsAdmin)
}
