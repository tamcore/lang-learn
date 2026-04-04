package api

import (
	"context"
	"time"

	"github.com/user/lang-learn/internal/models"
)

// mockUserStore implements store.UserStorer with configurable returns.
type mockUserStore struct {
	listFn          func(ctx context.Context) ([]models.User, error)
	getByIDFn       func(ctx context.Context, id string) (models.User, error)
	getByUsernameFn func(ctx context.Context, username string) (models.User, error)
	getByEmailFn    func(ctx context.Context, email string) (models.User, error)
	createFn        func(ctx context.Context, user models.User) error
	updateFn        func(ctx context.Context, user models.User) error
	deleteFn        func(ctx context.Context, id string) error
}

func (m *mockUserStore) List(ctx context.Context) ([]models.User, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockUserStore) GetByID(ctx context.Context, id string) (models.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return models.User{}, nil
}

func (m *mockUserStore) GetByUsername(ctx context.Context, username string) (models.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return models.User{}, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (models.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return models.User{}, nil
}

func (m *mockUserStore) Create(ctx context.Context, user models.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserStore) Update(ctx context.Context, user models.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// mockCourseStore implements store.CourseStorer with configurable returns.
type mockCourseStore struct {
	listFn    func(ctx context.Context) ([]models.Course, error)
	getByIDFn func(ctx context.Context, id string) (models.Course, error)
	createFn  func(ctx context.Context, course models.Course) error
	updateFn  func(ctx context.Context, course models.Course) error
	deleteFn  func(ctx context.Context, id string) error
}

func (m *mockCourseStore) List(ctx context.Context) ([]models.Course, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockCourseStore) GetByID(ctx context.Context, id string) (models.Course, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return models.Course{}, nil
}

func (m *mockCourseStore) Create(ctx context.Context, course models.Course) error {
	if m.createFn != nil {
		return m.createFn(ctx, course)
	}
	return nil
}

func (m *mockCourseStore) Update(ctx context.Context, course models.Course) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, course)
	}
	return nil
}

func (m *mockCourseStore) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// mockProgressStore implements store.ProgressStorer with configurable returns.
type mockProgressStore struct {
	getFn        func(ctx context.Context, userID, courseID string) (models.CourseProgress, error)
	upsertFn     func(ctx context.Context, progress models.CourseProgress) error
	listByUserFn func(ctx context.Context, userID string) ([]models.CourseProgress, error)
}

func (m *mockProgressStore) Get(ctx context.Context, userID, courseID string) (models.CourseProgress, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID, courseID)
	}
	return models.CourseProgress{}, nil
}

func (m *mockProgressStore) Upsert(ctx context.Context, progress models.CourseProgress) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, progress)
	}
	return nil
}

func (m *mockProgressStore) ListByUser(ctx context.Context, userID string) ([]models.CourseProgress, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID)
	}
	return nil, nil
}

// mockAuditStore implements store.AuditStorer with configurable returns.
type mockAuditStore struct {
	appendFn     func(ctx context.Context, entry models.AuditEntry) error
	listByDateFn func(ctx context.Context, date time.Time) ([]models.AuditEntry, error)
}

func (m *mockAuditStore) Append(ctx context.Context, entry models.AuditEntry) error {
	if m.appendFn != nil {
		return m.appendFn(ctx, entry)
	}
	return nil
}

func (m *mockAuditStore) ListByDate(ctx context.Context, date time.Time) ([]models.AuditEntry, error) {
	if m.listByDateFn != nil {
		return m.listByDateFn(ctx, date)
	}
	return nil, nil
}
