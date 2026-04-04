package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
)

func TestFileAuditStore_AppendAndListByDate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileAuditStore(dir)
	require.NoError(t, err)

	now := time.Now().UTC()
	entry := models.AuditEntry{
		ID:         "a1",
		Timestamp:  now,
		Action:     models.ActionUserCreated,
		ActorID:    "admin",
		TargetID:   "user1",
		TargetType: "user",
	}
	require.NoError(t, s.Append(context.Background(), entry))

	entries, err := s.ListByDate(context.Background(), now)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "a1", entries[0].ID)
}

func TestFileAuditStore_AppendMultiple(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileAuditStore(dir)
	require.NoError(t, err)

	now := time.Now().UTC()
	for i, id := range []string{"a1", "a2", "a3"} {
		entry := models.AuditEntry{
			ID:         id,
			Timestamp:  now,
			Action:     models.ActionUserCreated,
			ActorID:    "admin",
			TargetID:   "user" + id,
			TargetType: "user",
			Meta:       map[string]any{"index": i},
		}
		require.NoError(t, s.Append(context.Background(), entry))
	}

	entries, err := s.ListByDate(context.Background(), now)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestFileAuditStore_ListByDate_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileAuditStore(dir)
	require.NoError(t, err)

	entries, err := s.ListByDate(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFileAuditStore_DifferentDays(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s, err := store.NewFileAuditStore(dir)
	require.NoError(t, err)

	day1 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)

	require.NoError(t, s.Append(context.Background(), models.AuditEntry{
		ID: "a1", Timestamp: day1, Action: models.ActionUserCreated, ActorID: "x", TargetID: "y", TargetType: "user",
	}))
	require.NoError(t, s.Append(context.Background(), models.AuditEntry{
		ID: "a2", Timestamp: day2, Action: models.ActionUserDeleted, ActorID: "x", TargetID: "y", TargetType: "user",
	}))

	e1, err := s.ListByDate(context.Background(), day1)
	require.NoError(t, err)
	assert.Len(t, e1, 1)
	assert.Equal(t, "a1", e1[0].ID)

	e2, err := s.ListByDate(context.Background(), day2)
	require.NoError(t, err)
	assert.Len(t, e2, 1)
	assert.Equal(t, "a2", e2[0].ID)
}
