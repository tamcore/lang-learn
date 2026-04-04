// Package store defines the storage interfaces used throughout lang-learn.
// All concrete implementations must satisfy these interfaces, which enables
// handler tests to inject lightweight mocks without touching the filesystem.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/user/lang-learn/internal/models"
)

// Sentinel errors returned by all store implementations.
var (
	// ErrNotFound is returned when a requested record does not exist.
	ErrNotFound = errors.New("record not found")

	// ErrConflict is returned when a create/update violates a uniqueness
	// constraint (e.g. duplicate email address).
	ErrConflict = errors.New("record conflict")
)

// UserStorer defines CRUD operations for user accounts.
type UserStorer interface {
	// Create persists a new user. Returns ErrConflict if the email is taken.
	Create(ctx context.Context, user models.User) error

	// GetByID returns the user with the given ID. Returns ErrNotFound if absent.
	GetByID(ctx context.Context, id string) (models.User, error)

	// GetByEmail returns the user with the given email. Returns ErrNotFound if absent.
	GetByEmail(ctx context.Context, email string) (models.User, error)

	// Update overwrites the stored user record. Returns ErrNotFound if absent.
	Update(ctx context.Context, user models.User) error

	// Delete removes the user record. Returns ErrNotFound if absent.
	Delete(ctx context.Context, id string) error

	// List returns all user records in unspecified order.
	List(ctx context.Context) ([]models.User, error)
}

// CourseStorer defines CRUD operations for courses (including embedded lessons).
type CourseStorer interface {
	// Create persists a new course. Returns ErrConflict if the ID is taken.
	Create(ctx context.Context, course models.Course) error

	// GetByID returns the course with the given ID. Returns ErrNotFound if absent.
	GetByID(ctx context.Context, id string) (models.Course, error)

	// Update overwrites the stored course record. Returns ErrNotFound if absent.
	Update(ctx context.Context, course models.Course) error

	// Delete removes the course and its associated audio files. Returns ErrNotFound if absent.
	Delete(ctx context.Context, id string) error

	// List returns all courses in unspecified order (lessons included).
	List(ctx context.Context) ([]models.Course, error)
}

// ProgressStorer manages per-user progress records for each course.
type ProgressStorer interface {
	// Get returns the progress for a specific user/course pair. Returns ErrNotFound if absent.
	Get(ctx context.Context, userID, courseID string) (models.CourseProgress, error)

	// Upsert creates or replaces the progress record for the given user/course pair.
	Upsert(ctx context.Context, progress models.CourseProgress) error

	// ListByUser returns all progress records for the given user.
	ListByUser(ctx context.Context, userID string) ([]models.CourseProgress, error)
}

// AuditStorer manages the append-only audit log (daily JSON files).
type AuditStorer interface {
	// Append adds an entry to the audit log for the entry's date.
	Append(ctx context.Context, entry models.AuditEntry) error

	// ListByDate returns all audit entries recorded on the given calendar day.
	// The time portion of date is ignored; only year/month/day are used.
	ListByDate(ctx context.Context, date time.Time) ([]models.AuditEntry, error)
}
