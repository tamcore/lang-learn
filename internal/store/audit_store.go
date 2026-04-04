package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/user/lang-learn/internal/models"
)

// FileAuditStore implements AuditStorer using daily append-only JSON files.
// Each file is named {dir}/{YYYY-MM-DD}.json and contains a JSON array.
type FileAuditStore struct {
	dir string
	mu  sync.Mutex
}

// NewFileAuditStore creates a FileAuditStore rooted at dir.
func NewFileAuditStore(dir string) (*FileAuditStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("audit store: create directory: %w", err)
	}
	return &FileAuditStore{dir: dir}, nil
}

func (s *FileAuditStore) path(date time.Time) string {
	return filepath.Join(s.dir, date.Format("2006-01-02")+".json")
}

func (s *FileAuditStore) Append(_ context.Context, entry models.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.path(entry.Timestamp)

	var entries []models.AuditEntry
	data, err := os.ReadFile(p)
	if err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("audit store: parse existing: %w", err)
		}
	}

	entries = append(entries, entry)
	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("audit store: marshal: %w", err)
	}
	return os.WriteFile(p, out, 0o644)
}

func (s *FileAuditStore) ListByDate(_ context.Context, date time.Time) ([]models.AuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path(date))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no entries for this day
		}
		return nil, fmt.Errorf("audit store: read: %w", err)
	}
	var entries []models.AuditEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("audit store: parse: %w", err)
	}
	return entries, nil
}
