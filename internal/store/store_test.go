package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
)

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction checks.
// If the mock structs below do not implement the interfaces, this file will
// fail to compile, which is the desired signal.
// ---------------------------------------------------------------------------

// mockUserStorer satisfies store.UserStorer.
type mockUserStorer struct{}

func (m *mockUserStorer) Create(_ context.Context, _ models.User) error { return nil }
func (m *mockUserStorer) GetByID(_ context.Context, _ string) (models.User, error) {
	return models.User{}, nil
}
func (m *mockUserStorer) GetByEmail(_ context.Context, _ string) (models.User, error) {
	return models.User{}, nil
}
func (m *mockUserStorer) GetByUsername(_ context.Context, _ string) (models.User, error) {
	return models.User{}, nil
}
func (m *mockUserStorer) Update(_ context.Context, _ models.User) error { return nil }
func (m *mockUserStorer) Delete(_ context.Context, _ string) error      { return nil }
func (m *mockUserStorer) List(_ context.Context) ([]models.User, error) { return nil, nil }

var _ store.UserStorer = (*mockUserStorer)(nil)

// mockCourseStorer satisfies store.CourseStorer.
type mockCourseStorer struct{}

func (m *mockCourseStorer) Create(_ context.Context, _ models.Course) error { return nil }
func (m *mockCourseStorer) GetByID(_ context.Context, _ string) (models.Course, error) {
	return models.Course{}, nil
}
func (m *mockCourseStorer) Update(_ context.Context, _ models.Course) error { return nil }
func (m *mockCourseStorer) Delete(_ context.Context, _ string) error        { return nil }
func (m *mockCourseStorer) List(_ context.Context) ([]models.Course, error) { return nil, nil }

var _ store.CourseStorer = (*mockCourseStorer)(nil)

// mockProgressStorer satisfies store.ProgressStorer.
type mockProgressStorer struct{}

func (m *mockProgressStorer) Get(_ context.Context, _, _ string) (models.CourseProgress, error) {
	return models.CourseProgress{}, nil
}
func (m *mockProgressStorer) Upsert(_ context.Context, _ models.CourseProgress) error { return nil }
func (m *mockProgressStorer) ListByUser(_ context.Context, _ string) ([]models.CourseProgress, error) {
	return nil, nil
}

var _ store.ProgressStorer = (*mockProgressStorer)(nil)

// mockAuditStorer satisfies store.AuditStorer.
type mockAuditStorer struct{}

func (m *mockAuditStorer) Append(_ context.Context, _ models.AuditEntry) error { return nil }
func (m *mockAuditStorer) ListByDate(_ context.Context, _ time.Time) ([]models.AuditEntry, error) {
	return nil, nil
}

var _ store.AuditStorer = (*mockAuditStorer)(nil)

// ---------------------------------------------------------------------------
// Sentinel error tests.
// ---------------------------------------------------------------------------

func TestErrNotFound_IsItself(t *testing.T) {
	t.Parallel()
	assert.True(t, errors.Is(store.ErrNotFound, store.ErrNotFound))
}

func TestErrConflict_IsItself(t *testing.T) {
	t.Parallel()
	assert.True(t, errors.Is(store.ErrConflict, store.ErrConflict))
}

func TestErrNotFound_ErrorMessage(t *testing.T) {
	t.Parallel()
	assert.Contains(t, store.ErrNotFound.Error(), "not found")
}

func TestErrConflict_ErrorMessage(t *testing.T) {
	t.Parallel()
	assert.Contains(t, store.ErrConflict.Error(), "conflict")
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	t.Parallel()
	assert.False(t, errors.Is(store.ErrNotFound, store.ErrConflict))
	assert.False(t, errors.Is(store.ErrConflict, store.ErrNotFound))
}

func TestErrNotFound_WrappedWithErrors(t *testing.T) {
	t.Parallel()
	wrapped := errors.Join(store.ErrNotFound, errors.New("context info"))
	assert.True(t, errors.Is(wrapped, store.ErrNotFound))
}

func TestErrConflict_WrappedWithFmt(t *testing.T) {
	t.Parallel()
	// Sentinel can be wrapped using standard wrapping patterns
	wrapped := errors.Join(store.ErrConflict, errors.New("email already exists"))
	assert.True(t, errors.Is(wrapped, store.ErrConflict))
}
