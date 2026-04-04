package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/user/lang-learn/internal/models"
)

// FileProgressStore implements ProgressStorer using one JSON file per user/course pair.
// Files are stored as {dir}/{userID}/{courseID}.json.
type FileProgressStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileProgressStore creates a FileProgressStore rooted at dir.
func NewFileProgressStore(dir string) (*FileProgressStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("progress store: create directory: %w", err)
	}
	return &FileProgressStore{dir: dir}, nil
}

func (s *FileProgressStore) path(userID, courseID string) string {
	return filepath.Join(s.dir, userID, courseID+".json")
}

func (s *FileProgressStore) Get(_ context.Context, userID, courseID string) (models.CourseProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path(userID, courseID))
	if err != nil {
		if os.IsNotExist(err) {
			return models.CourseProgress{}, fmt.Errorf("progress %s/%s: %w", userID, courseID, ErrNotFound)
		}
		return models.CourseProgress{}, fmt.Errorf("progress store: read: %w", err)
	}
	var p models.CourseProgress
	if err := json.Unmarshal(data, &p); err != nil {
		return models.CourseProgress{}, fmt.Errorf("progress store: parse: %w", err)
	}
	return p, nil
}

func (s *FileProgressStore) Upsert(_ context.Context, progress models.CourseProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userDir := filepath.Join(s.dir, progress.UserID)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return fmt.Errorf("progress store: create user dir: %w", err)
	}

	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return fmt.Errorf("progress store: marshal: %w", err)
	}
	return os.WriteFile(s.path(progress.UserID, progress.CourseID), data, 0o644)
}

func (s *FileProgressStore) ListByUser(_ context.Context, userID string) ([]models.CourseProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userDir := filepath.Join(s.dir, userID)
	matches, err := filepath.Glob(filepath.Join(userDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("progress store: list: %w", err)
	}

	var results []models.CourseProgress
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var p models.CourseProgress
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		results = append(results, p)
	}
	return results, nil
}
