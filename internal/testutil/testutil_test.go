package testutil_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/testutil"
)

// --- MakeUser ---

func TestMakeUser_Defaults(t *testing.T) {
	t.Parallel()
	u := testutil.MakeUser()

	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.Username)
	assert.NotEmpty(t, u.Email)
	assert.NotEmpty(t, u.PasswordHash)
	assert.False(t, u.IsAdmin)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
}

func TestMakeUser_Override(t *testing.T) {
	t.Parallel()
	u := testutil.MakeUser(func(u *models.User) {
		u.Username = "alice"
		u.IsAdmin = true
	})

	assert.Equal(t, "alice", u.Username)
	assert.True(t, u.IsAdmin)
	// Unchanged fields keep defaults
	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.Email)
}

func TestMakeUser_MultipleOverrides(t *testing.T) {
	t.Parallel()
	u := testutil.MakeUser(
		func(u *models.User) { u.Username = "bob" },
		func(u *models.User) { u.Email = "bob@example.com" },
	)

	assert.Equal(t, "bob", u.Username)
	assert.Equal(t, "bob@example.com", u.Email)
}

func TestMakeUser_EachCallUnique(t *testing.T) {
	t.Parallel()
	u1 := testutil.MakeUser()
	u2 := testutil.MakeUser()

	// Each call should produce a distinct user (unique IDs)
	assert.NotEqual(t, u1.ID, u2.ID)
}

// --- MakeCourse ---

func TestMakeCourse_Defaults(t *testing.T) {
	t.Parallel()
	c := testutil.MakeCourse()

	assert.NotEmpty(t, c.ID)
	assert.NotEmpty(t, c.Title)
	assert.NotEmpty(t, c.SourceLang)
	assert.NotEmpty(t, c.TargetLang)
	assert.Equal(t, models.DirectionForward, c.Direction)
	assert.Equal(t, models.PerspectiveMale, c.Perspective)
	assert.NotEmpty(t, c.BlueprintID)
	assert.False(t, c.CreatedAt.IsZero())
	assert.NotNil(t, c.Lessons)
}

func TestMakeCourse_Override(t *testing.T) {
	t.Parallel()
	c := testutil.MakeCourse(func(c *models.Course) {
		c.TargetLang = "fr"
		c.Direction = models.DirectionReverse
		c.Perspective = models.PerspectiveFemale
	})

	assert.Equal(t, "fr", c.TargetLang)
	assert.Equal(t, models.DirectionReverse, c.Direction)
	assert.Equal(t, models.PerspectiveFemale, c.Perspective)
	assert.NotEmpty(t, c.ID)
}

func TestMakeCourse_EachCallUnique(t *testing.T) {
	t.Parallel()
	c1 := testutil.MakeCourse()
	c2 := testutil.MakeCourse()

	assert.NotEqual(t, c1.ID, c2.ID)
}

// --- WithJWT ---

func TestWithJWT_SetsAuthorizationHeader(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r = testutil.WithJWT(t, r, "testsecret", "user-123", false)

	auth := r.Header.Get("Authorization")
	assert.True(t, strings.HasPrefix(auth, "Bearer "), "header should start with 'Bearer '")
}

func TestWithJWT_TokenIsParseable(t *testing.T) {
	t.Parallel()
	secret := "my-test-secret"
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r = testutil.WithJWT(t, r, secret, "user-456", true)

	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	require.NoError(t, err)
	assert.True(t, token.Valid)
}

func TestWithJWT_ClaimsContainUserID(t *testing.T) {
	t.Parallel()
	secret := "claim-secret"
	userID := "user-789"
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r = testutil.WithJWT(t, r, secret, userID, false)

	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, userID, claims["sub"])
}

func TestWithJWT_ClaimsContainIsAdmin(t *testing.T) {
	t.Parallel()
	secret := "admin-secret"
	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = testutil.WithJWT(t, r, secret, "admin-1", true)

	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, true, claims["is_admin"])
}

func TestWithJWT_TokenNotExpired(t *testing.T) {
	t.Parallel()
	secret := "expiry-secret"
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r = testutil.WithJWT(t, r, secret, "user-exp", false)

	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

	require.NoError(t, err, "token should not be expired")
}

// --- TempDataDir ---

func TestTempDataDir_ReturnsExistingDir(t *testing.T) {
	t.Parallel()
	dir := testutil.TempDataDir(t)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestTempDataDir_CreatesSubdirectories(t *testing.T) {
	t.Parallel()
	dir := testutil.TempDataDir(t)

	for _, sub := range []string{"users", "courses", "progress", "audit"} {
		path := filepath.Join(dir, sub)
		info, err := os.Stat(path)
		require.NoError(t, err, "subdir %q should exist", sub)
		assert.True(t, info.IsDir(), "subdir %q should be a directory", sub)
	}
}

func TestTempDataDir_EachCallReturnsUniqueDir(t *testing.T) {
	t.Parallel()
	dir1 := testutil.TempDataDir(t)
	dir2 := testutil.TempDataDir(t)

	assert.NotEqual(t, dir1, dir2)
}
