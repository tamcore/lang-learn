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

// FileCourseStore implements CourseStorer using one JSON file per course.
// Each course is stored as {dir}/{courseID}/course.json.
type FileCourseStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileCourseStore creates a FileCourseStore rooted at dir.
func NewFileCourseStore(dir string) (*FileCourseStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("course store: create directory: %w", err)
	}
	return &FileCourseStore{dir: dir}, nil
}

func (s *FileCourseStore) courseDir(id string) string {
	return filepath.Join(s.dir, id)
}

func (s *FileCourseStore) coursePath(id string) string {
	return filepath.Join(s.dir, id, "course.json")
}

func (s *FileCourseStore) Create(_ context.Context, course models.Course) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.courseDir(course.ID)
	if _, err := os.Stat(s.coursePath(course.ID)); err == nil {
		return fmt.Errorf("course id %q already exists: %w", course.ID, ErrConflict)
	}

	if err := os.MkdirAll(filepath.Join(dir, "audio"), 0o755); err != nil {
		return fmt.Errorf("course store: create dirs: %w", err)
	}

	return s.writeJSON(course)
}

func (s *FileCourseStore) GetByID(_ context.Context, id string) (models.Course, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readCourse(id)
}

func (s *FileCourseStore) Update(_ context.Context, course models.Course) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.coursePath(course.ID)); os.IsNotExist(err) {
		return fmt.Errorf("course id %q: %w", course.ID, ErrNotFound)
	}
	return s.writeJSON(course)
}

func (s *FileCourseStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.courseDir(id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("course id %q: %w", id, ErrNotFound)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("course store: delete course %q: %w", id, err)
	}
	return nil
}

func (s *FileCourseStore) List(_ context.Context) ([]models.Course, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("course store: list dir: %w", err)
	}

	var courses []models.Course
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c, err := s.readCourse(e.Name())
		if err != nil {
			continue // skip corrupt entries
		}
		courses = append(courses, c)
	}
	return courses, nil
}

// AudioDir returns the audio directory path for a course.
func (s *FileCourseStore) AudioDir(courseID string) string {
	return filepath.Join(s.courseDir(courseID), "audio")
}

func (s *FileCourseStore) readCourse(id string) (models.Course, error) {
	data, err := os.ReadFile(s.coursePath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return models.Course{}, fmt.Errorf("course id %q: %w", id, ErrNotFound)
		}
		return models.Course{}, fmt.Errorf("course store: read %q: %w", id, err)
	}
	var c models.Course
	if err := json.Unmarshal(data, &c); err != nil {
		return models.Course{}, fmt.Errorf("course store: parse %q: %w", id, err)
	}
	return c, nil
}

func (s *FileCourseStore) writeJSON(course models.Course) error {
	data, err := json.MarshalIndent(course, "", "  ")
	if err != nil {
		return fmt.Errorf("course store: marshal %q: %w", course.ID, err)
	}
	return os.WriteFile(s.coursePath(course.ID), data, 0o644)
}
