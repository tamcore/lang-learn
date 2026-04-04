package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
)

func TestNewRouter_Healthz(t *testing.T) {
	t.Parallel()
	dir := testutil.TempDataDir(t)

	users, err := store.NewFileUserStore(dir + "/users")
	require.NoError(t, err)
	courses, err := store.NewFileCourseStore(dir + "/courses")
	require.NoError(t, err)
	progress, err := store.NewFileProgressStore(dir + "/progress")
	require.NoError(t, err)
	audit, err := store.NewFileAuditStore(dir + "/audit")
	require.NoError(t, err)

	router := NewRouter(RouterConfig{
		JWTSecret:  testSecret,
		Users:      users,
		Courses:    courses,
		Progress:   progress,
		Audit:      audit,
		CoursesDir: dir + "/courses",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		BcryptCost: 4,
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, "ok", data["status"])
}
