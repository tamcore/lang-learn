package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newUserStore(t *testing.T) *store.FileUserStore {
	t.Helper()
	dir := testutil.TempDataDir(t)
	s, err := store.NewFileUserStore(filepath.Join(dir, "users"))
	require.NoError(t, err)
	return s
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewFileUserStore_CreatesDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	usersDir := filepath.Join(dir, "users", "nested")
	s, err := store.NewFileUserStore(usersDir)
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestNewFileUserStore_ExistingDir(t *testing.T) {
	t.Parallel()
	// Should succeed even when directory already exists.
	dir := testutil.TempDataDir(t)
	s, err := store.NewFileUserStore(filepath.Join(dir, "users"))
	require.NoError(t, err)
	assert.NotNil(t, s)
}

// ---------------------------------------------------------------------------
// Interface satisfaction (compile-time guard)
// ---------------------------------------------------------------------------

func TestFileUserStore_ImplementsUserStorer(t *testing.T) {
	t.Parallel()
	var _ store.UserStorer = (*store.FileUserStore)(nil)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestFileUserStore_Create_Success(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()

	err := s.Create(context.Background(), u)
	require.NoError(t, err)
}

func TestFileUserStore_Create_PersistedDataMatchesInput(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	got, err := s.GetByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, u, got)
}

func TestFileUserStore_Create_DuplicateID(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	// Different email, same ID → ErrConflict
	dup := testutil.MakeUser(func(x *models.User) { x.ID = u.ID })
	err := s.Create(context.Background(), dup)
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrConflict), "expected ErrConflict, got %v", err)
}

func TestFileUserStore_Create_DuplicateUsername(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	// Different ID, same username → ErrConflict
	dup := testutil.MakeUser(func(x *models.User) { x.Username = u.Username })
	err := s.Create(context.Background(), dup)
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrConflict), "expected ErrConflict, got %v", err)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestFileUserStore_GetByID_Found(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	got, err := s.GetByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, u, got)
}

func TestFileUserStore_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	_, err := s.GetByID(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected ErrNotFound, got %v", err)
}

// ---------------------------------------------------------------------------
// GetByEmail
// ---------------------------------------------------------------------------

func TestFileUserStore_GetByEmail_Found(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	got, err := s.GetByEmail(context.Background(), u.Email)
	require.NoError(t, err)
	assert.Equal(t, u, got)
}

func TestFileUserStore_GetByEmail_NotFound(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	_, err := s.GetByEmail(context.Background(), "nobody@example.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected ErrNotFound, got %v", err)
}

func TestFileUserStore_GetByEmail_IgnoresOtherUsers(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u1 := testutil.MakeUser()
	u2 := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u1))
	require.NoError(t, s.Create(context.Background(), u2))

	got, err := s.GetByEmail(context.Background(), u2.Email)
	require.NoError(t, err)
	assert.Equal(t, u2, got)
}

// ---------------------------------------------------------------------------
// GetByUsername
// ---------------------------------------------------------------------------

func TestFileUserStore_GetByUsername_Found(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	got, err := s.GetByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	assert.Equal(t, u, got)
}

func TestFileUserStore_GetByUsername_NotFound(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	_, err := s.GetByUsername(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected ErrNotFound, got %v", err)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestFileUserStore_Update_Success(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	u.Username = "updated-username"
	u.IsAdmin = true
	require.NoError(t, s.Update(context.Background(), u))

	got, err := s.GetByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-username", got.Username)
	assert.True(t, got.IsAdmin)
}

func TestFileUserStore_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()

	err := s.Update(context.Background(), u)
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected ErrNotFound, got %v", err)
}

func TestFileUserStore_Update_UsernameConflictWithOtherUser(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u1 := testutil.MakeUser()
	u2 := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u1))
	require.NoError(t, s.Create(context.Background(), u2))

	// Try to assign u1's username to u2
	u2.Username = u1.Username
	err := s.Update(context.Background(), u2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrConflict), "expected ErrConflict, got %v", err)
}

func TestFileUserStore_Update_SameUsernameAllowed(t *testing.T) {
	t.Parallel()
	// Updating a user without changing the username must succeed.
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	u.Username = "new-name"
	err := s.Update(context.Background(), u) // email unchanged
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestFileUserStore_Delete_Success(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	require.NoError(t, s.Delete(context.Background(), u.ID))

	_, err := s.GetByID(context.Background(), u.ID)
	assert.True(t, errors.Is(err, store.ErrNotFound), "user should be gone")
}

func TestFileUserStore_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	err := s.Delete(context.Background(), "ghost-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrNotFound), "expected ErrNotFound, got %v", err)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestFileUserStore_List_Empty(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	users, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestFileUserStore_List_Single(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	users, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, u, users[0])
}

func TestFileUserStore_List_Multiple(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u1 := testutil.MakeUser()
	u2 := testutil.MakeUser()
	u3 := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u1))
	require.NoError(t, s.Create(context.Background(), u2))
	require.NoError(t, s.Create(context.Background(), u3))

	users, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 3)

	ids := make(map[string]bool, 3)
	for _, u := range users {
		ids[u.ID] = true
	}
	assert.True(t, ids[u1.ID])
	assert.True(t, ids[u2.ID])
	assert.True(t, ids[u3.ID])
}

func TestFileUserStore_List_AfterDelete(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)
	u1 := testutil.MakeUser()
	u2 := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u1))
	require.NoError(t, s.Create(context.Background(), u2))
	require.NoError(t, s.Delete(context.Background(), u1.ID))

	users, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, u2.ID, users[0].ID)
}

// ---------------------------------------------------------------------------
// Error paths (coverage)
// ---------------------------------------------------------------------------

// newUserStoreAt creates a store inside a specific directory, returning
// both the store and the users sub-directory path for low-level manipulation.
func newUserStoreAt(t *testing.T) (*store.FileUserStore, string) {
	t.Helper()
	dir := testutil.TempDataDir(t)
	usersDir := filepath.Join(dir, "users")
	s, err := store.NewFileUserStore(usersDir)
	require.NoError(t, err)
	return s, usersDir
}

func TestNewFileUserStore_FailsWhenPathIsFile(t *testing.T) {
	t.Parallel()
	// Create a regular file where MkdirAll would need to create a directory.
	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	_, err := store.NewFileUserStore(filepath.Join(blocker, "users"))
	require.Error(t, err, "should fail when a file blocks directory creation")
}

func TestFileUserStore_GetByID_CorruptJSON(t *testing.T) {
	t.Parallel()
	s, usersDir := newUserStoreAt(t)
	// Write a file whose name matches a valid user ID but contains invalid JSON.
	require.NoError(t, os.WriteFile(
		filepath.Join(usersDir, "bad-user.json"),
		[]byte("{not valid json"),
		0o644,
	))

	_, err := s.GetByID(context.Background(), "bad-user")
	require.Error(t, err, "should return error for corrupt JSON")
}

func TestFileUserStore_List_CorruptJSON(t *testing.T) {
	t.Parallel()
	s, usersDir := newUserStoreAt(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(usersDir, "corrupt.json"),
		[]byte("!!!"),
		0o644,
	))

	_, err := s.List(context.Background())
	require.Error(t, err, "List should propagate readAll errors")
}

func TestFileUserStore_GetByEmail_CorruptJSON(t *testing.T) {
	t.Parallel()
	s, usersDir := newUserStoreAt(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(usersDir, "corrupt.json"),
		[]byte("!!!"),
		0o644,
	))

	_, err := s.GetByEmail(context.Background(), "anyone@example.com")
	require.Error(t, err, "GetByEmail should propagate readAll errors")
}

func TestFileUserStore_Create_CorruptExistingFile(t *testing.T) {
	t.Parallel()
	s, usersDir := newUserStoreAt(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(usersDir, "corrupt.json"),
		[]byte("!!!"),
		0o644,
	))

	u := testutil.MakeUser()
	err := s.Create(context.Background(), u)
	require.Error(t, err, "Create should propagate readAll errors during email scan")
}

func TestFileUserStore_Update_CorruptExistingFile(t *testing.T) {
	t.Parallel()
	s, usersDir := newUserStoreAt(t)

	// Create a valid user first (so update has something to find)
	u := testutil.MakeUser()
	require.NoError(t, s.Create(context.Background(), u))

	// Now corrupt a different file in the same dir
	require.NoError(t, os.WriteFile(
		filepath.Join(usersDir, "corrupt.json"),
		[]byte("!!!"),
		0o644,
	))

	u.Username = "changed"
	err := s.Update(context.Background(), u)
	require.Error(t, err, "Update should propagate readAll errors during email scan")
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestFileUserStore_Concurrent_CreateAndRead(t *testing.T) {
	t.Parallel()
	s := newUserStore(t)

	const n = 10
	users := make([]models.User, n)
	for i := range users {
		users[i] = testutil.MakeUser()
	}

	// Create all users concurrently
	errCh := make(chan error, n)
	for _, u := range users {
		go func() {
			errCh <- s.Create(context.Background(), u)
		}()
	}
	for range users {
		require.NoError(t, <-errCh)
	}

	// Verify all are retrievable
	list, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, n)
}
