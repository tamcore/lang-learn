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

// FileUserStore implements UserStorer using one JSON file per user.
// Files are stored as {dir}/{userID}.json.
// All operations are protected by a read-write mutex for safe concurrent use.
type FileUserStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileUserStore creates a FileUserStore rooted at dir.
// The directory is created if it does not exist.
func NewFileUserStore(dir string) (*FileUserStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("user store: create directory: %w", err)
	}
	return &FileUserStore{dir: dir}, nil
}

// path returns the file path for a given user ID.
func (s *FileUserStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// readAll reads every *.json file in the store directory.
// Caller must hold at least a read lock.
func (s *FileUserStore) readAll() ([]models.User, error) {
	matches, err := filepath.Glob(filepath.Join(s.dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("user store: list files: %w", err)
	}
	users := make([]models.User, 0, len(matches))
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("user store: read %s: %w", m, err)
		}
		var u models.User
		if err := json.Unmarshal(data, &u); err != nil {
			return nil, fmt.Errorf("user store: parse %s: %w", m, err)
		}
		users = append(users, u)
	}
	return users, nil
}

// write marshals user and atomically writes it to {dir}/{id}.json.
// Caller must hold the write lock.
func (s *FileUserStore) write(user models.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("user store: marshal user %q: %w", user.ID, err)
	}
	if err := os.WriteFile(s.path(user.ID), data, 0o644); err != nil {
		return fmt.Errorf("user store: write user %q: %w", user.ID, err)
	}
	return nil
}

// Create persists a new user. Returns ErrConflict if the ID or username is already taken.
func (s *FileUserStore) Create(_ context.Context, user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Duplicate ID check — fast path before full scan.
	if _, err := os.Stat(s.path(user.ID)); err == nil {
		return fmt.Errorf("user id %q already exists: %w", user.ID, ErrConflict)
	}

	// Duplicate username check requires scanning all existing users.
	existing, err := s.readAll()
	if err != nil {
		return err
	}
	for _, u := range existing {
		if u.Username == user.Username {
			return fmt.Errorf("username %q already taken: %w", user.Username, ErrConflict)
		}
	}

	return s.write(user)
}

// GetByID returns the user with the given ID. Returns ErrNotFound if absent.
func (s *FileUserStore) GetByID(_ context.Context, id string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return models.User{}, fmt.Errorf("user id %q: %w", id, ErrNotFound)
		}
		return models.User{}, fmt.Errorf("user store: read user %q: %w", id, err)
	}

	var u models.User
	if err := json.Unmarshal(data, &u); err != nil {
		return models.User{}, fmt.Errorf("user store: parse user %q: %w", id, err)
	}
	return u, nil
}

// GetByEmail returns the user with the given email. Returns ErrNotFound if absent.
func (s *FileUserStore) GetByEmail(_ context.Context, email string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users, err := s.readAll()
	if err != nil {
		return models.User{}, err
	}
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return models.User{}, fmt.Errorf("email %q: %w", email, ErrNotFound)
}

// GetByUsername returns the user with the given username. Returns ErrNotFound if absent.
func (s *FileUserStore) GetByUsername(_ context.Context, username string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users, err := s.readAll()
	if err != nil {
		return models.User{}, err
	}
	for _, u := range users {
		if u.Username == username {
			return u, nil
		}
	}
	return models.User{}, fmt.Errorf("username %q: %w", username, ErrNotFound)
}

// Update overwrites the stored user record. Returns ErrNotFound if absent.
// Returns ErrConflict if the new username is already taken by a different user.
func (s *FileUserStore) Update(_ context.Context, user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path(user.ID)); os.IsNotExist(err) {
		return fmt.Errorf("user id %q: %w", user.ID, ErrNotFound)
	}

	// Username uniqueness check against all other users.
	existing, err := s.readAll()
	if err != nil {
		return err
	}
	for _, u := range existing {
		if u.Username == user.Username && u.ID != user.ID {
			return fmt.Errorf("username %q already taken: %w", user.Username, ErrConflict)
		}
	}

	return s.write(user)
}

// Delete removes the user record. Returns ErrNotFound if absent.
func (s *FileUserStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("user id %q: %w", id, ErrNotFound)
		}
		return fmt.Errorf("user store: delete user %q: %w", id, err)
	}
	return nil
}

// List returns all user records in unspecified order.
func (s *FileUserStore) List(_ context.Context) ([]models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users, err := s.readAll()
	if err != nil {
		return nil, err
	}
	return users, nil
}
