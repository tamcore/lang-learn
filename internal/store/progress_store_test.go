package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
)

func TestFileProgressStore_UpsertAndGet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileProgressStore(dir)
	require.NoError(t, err)

	p := makeProgress("user1", "course1")
	require.NoError(t, s.Upsert(context.Background(), p))

	got, err := s.Get(context.Background(), "user1", "course1")
	require.NoError(t, err)
	assert.Equal(t, p.CurrentLesson, got.CurrentLesson)
}

func TestFileProgressStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileProgressStore(dir)
	require.NoError(t, err)

	_, err = s.Get(context.Background(), "user1", "nonexistent")
	assert.True(t, errors.Is(err, store.ErrNotFound))
}

func TestFileProgressStore_Upsert_Overwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileProgressStore(dir)
	require.NoError(t, err)

	p := makeProgress("user1", "course1")
	require.NoError(t, s.Upsert(context.Background(), p))

	p.CurrentLesson = 5
	require.NoError(t, s.Upsert(context.Background(), p))

	got, err := s.Get(context.Background(), "user1", "course1")
	require.NoError(t, err)
	assert.Equal(t, 5, got.CurrentLesson)
}

func TestFileProgressStore_ListByUser(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileProgressStore(dir)
	require.NoError(t, err)

	require.NoError(t, s.Upsert(context.Background(), makeProgress("user1", "c1")))
	require.NoError(t, s.Upsert(context.Background(), makeProgress("user1", "c2")))
	require.NoError(t, s.Upsert(context.Background(), makeProgress("user2", "c1")))

	list, err := s.ListByUser(context.Background(), "user1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestFileProgressStore_ListByUser_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileProgressStore(dir)
	require.NoError(t, err)

	list, err := s.ListByUser(context.Background(), "user1")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func makeProgress(userID, courseID string) models.CourseProgress {
	now := time.Now().UTC().Truncate(time.Second)
	return models.CourseProgress{
		UserID:           userID,
		CourseID:         courseID,
		CurrentLesson:    1,
		LessonsCompleted: nil,
		LastAccessedAt:   now,
		UpdatedAt:        now,
	}
}
