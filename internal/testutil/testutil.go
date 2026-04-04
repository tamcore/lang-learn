// Package testutil provides shared test helpers for the lang-learn test suite.
// Helpers in this package are NOT for production use.
package testutil

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
)

// counter generates unique numeric suffixes for test fixtures.
var counter atomic.Int64

func next() int64 {
	return counter.Add(1)
}

// MakeUser returns a models.User with sensible defaults for tests.
// Call optional override functions to customise specific fields.
func MakeUser(overrides ...func(*models.User)) models.User {
	n := next()
	now := time.Now().UTC().Truncate(time.Second)
	u := models.User{
		ID:           fmt.Sprintf("user-%d", n),
		Username:     fmt.Sprintf("testuser%d", n),
		Email:        fmt.Sprintf("testuser%d@example.com", n),
		PasswordHash: "$2a$12$testonlyhashvalue",
		IsAdmin:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	for _, o := range overrides {
		o(&u)
	}
	return u
}

// MakeCourse returns a models.Course with sensible defaults for tests.
// Call optional override functions to customise specific fields.
func MakeCourse(overrides ...func(*models.Course)) models.Course {
	n := next()
	now := time.Now().UTC().Truncate(time.Second)
	c := models.Course{
		ID:          fmt.Sprintf("course-%d", n),
		Title:       fmt.Sprintf("Test Course %d", n),
		Description: "A course created for testing",
		SourceLang:  "en",
		TargetLang:  "es",
		Direction:   models.DirectionForward,
		Perspective: models.PerspectiveMale,
		BlueprintID: "travel-basics-v1",
		LessonCount: 1,
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: "test-admin",
		Lessons:     []models.Lesson{},
	}
	for _, o := range overrides {
		o(&c)
	}
	return c
}

// WithJWT signs an HS256 JWT containing the given userID and isAdmin claims,
// sets "Authorization: Bearer <token>" on r, and returns the modified request.
// The token expires in 15 minutes from the time of creation.
func WithJWT(t *testing.T, r *http.Request, secret, userID string, isAdmin bool) *http.Request {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":      userID,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err, "WithJWT: failed to sign token")

	r.Header.Set("Authorization", "Bearer "+signed)
	return r
}

// TempDataDir creates a temporary directory with the standard data sub-directories
// (users, courses, progress, audit) and registers cleanup with t.Cleanup.
// It uses t.TempDir() so the directory is removed automatically after the test.
func TempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"users", "courses", "progress", "audit"} {
		err := os.MkdirAll(filepath.Join(dir, sub), 0o755)
		require.NoError(t, err, "TempDataDir: failed to create subdir %q", sub)
	}
	return dir
}
