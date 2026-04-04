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

func TestFileCourseStore_CreateAndGetByID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	c := makeCourse("c1")
	require.NoError(t, s.Create(context.Background(), c))

	got, err := s.GetByID(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, c.Title, got.Title)
	assert.Equal(t, c.SourceLang, got.SourceLang)
}

func TestFileCourseStore_CreateDuplicate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	c := makeCourse("c1")
	require.NoError(t, s.Create(context.Background(), c))
	err = s.Create(context.Background(), c)
	assert.True(t, errors.Is(err, store.ErrConflict))
}

func TestFileCourseStore_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	_, err = s.GetByID(context.Background(), "nonexistent")
	assert.True(t, errors.Is(err, store.ErrNotFound))
}

func TestFileCourseStore_Update(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	c := makeCourse("c1")
	require.NoError(t, s.Create(context.Background(), c))

	c.Title = "Updated"
	require.NoError(t, s.Update(context.Background(), c))

	got, err := s.GetByID(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
}

func TestFileCourseStore_Update_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	err = s.Update(context.Background(), makeCourse("nope"))
	assert.True(t, errors.Is(err, store.ErrNotFound))
}

func TestFileCourseStore_Delete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	c := makeCourse("c1")
	require.NoError(t, s.Create(context.Background(), c))
	require.NoError(t, s.Delete(context.Background(), "c1"))

	_, err = s.GetByID(context.Background(), "c1")
	assert.True(t, errors.Is(err, store.ErrNotFound))
}

func TestFileCourseStore_Delete_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	err = s.Delete(context.Background(), "nope")
	assert.True(t, errors.Is(err, store.ErrNotFound))
}

func TestFileCourseStore_List(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	require.NoError(t, s.Create(context.Background(), makeCourse("c1")))
	require.NoError(t, s.Create(context.Background(), makeCourse("c2")))

	list, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestFileCourseStore_List_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileCourseStore(dir)
	require.NoError(t, err)

	list, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func makeCourse(id string) models.Course {
	now := time.Now().UTC().Truncate(time.Second)
	return models.Course{
		ID:          id,
		Title:       "Test Course " + id,
		Description: "Test",
		SourceLang:  "en",
		TargetLang:  "sk",
		Direction:   models.DirectionForward,
		Perspective: models.PerspectiveMale,
		BlueprintID: "travel-basics-v1",
		LessonCount: 1,
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: "test",
		Lessons:     []models.Lesson{},
	}
}
