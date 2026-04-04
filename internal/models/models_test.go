package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
)

func TestCourseDirectionConstants(t *testing.T) {
	assert.Equal(t, models.CourseDirection("forward"), models.DirectionForward)
	assert.Equal(t, models.CourseDirection("reverse"), models.DirectionReverse)
}

func TestPerspectiveConstants(t *testing.T) {
	assert.Equal(t, models.Perspective("male"), models.PerspectiveMale)
	assert.Equal(t, models.Perspective("female"), models.PerspectiveFemale)
}

func TestTurnSpeakerConstants(t *testing.T) {
	assert.Equal(t, models.TurnSpeaker("system"), models.SpeakerSystem)
	assert.Equal(t, models.TurnSpeaker("user"), models.SpeakerUser)
}

func TestAuditActionConstants(t *testing.T) {
	assert.Equal(t, models.AuditAction("user.created"), models.ActionUserCreated)
	assert.Equal(t, models.AuditAction("user.deleted"), models.ActionUserDeleted)
	assert.Equal(t, models.AuditAction("course.generated"), models.ActionCourseGenerated)
	assert.Equal(t, models.AuditAction("course.deleted"), models.ActionCourseDeleted)
	assert.Equal(t, models.AuditAction("admin.login"), models.ActionAdminLogin)
	assert.Equal(t, models.AuditAction("speaking.evaluated"), models.ActionSpeakingEvaluated)
}

func TestUser_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	u := models.User{
		ID:           "abc-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$12$hash",
		IsAdmin:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(u)
	require.NoError(t, err)

	var got models.User
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, u, got)
}

func TestUser_JSONFieldNames(t *testing.T) {
	t.Parallel()
	u := models.User{ID: "x", Username: "u", Email: "e@e.com"}
	data, err := json.Marshal(u)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "username")
	assert.Contains(t, m, "email")
	assert.Contains(t, m, "password_hash")
	assert.Contains(t, m, "is_admin")
	assert.Contains(t, m, "created_at")
	assert.Contains(t, m, "updated_at")
}

func TestTurn_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	turn := models.Turn{
		ID:           "turn-1",
		Sequence:     1,
		Speaker:      models.SpeakerSystem,
		Text:         "Hello",
		Translation:  "Hola",
		AudioFile:    "audio/turn-1.mp3",
		IsBlurred:    false,
		SpacedRepeat: true,
		DelayAfterMs: 500,
	}

	data, err := json.Marshal(turn)
	require.NoError(t, err)

	var got models.Turn
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, turn, got)
}

func TestTurn_JSONFieldNames(t *testing.T) {
	t.Parallel()
	turn := models.Turn{Speaker: models.SpeakerUser, IsBlurred: true}
	data, err := json.Marshal(turn)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "sequence")
	assert.Contains(t, m, "speaker")
	assert.Contains(t, m, "text")
	assert.Contains(t, m, "translation")
	assert.Contains(t, m, "audio_file")
	assert.Contains(t, m, "is_blurred")
	assert.Contains(t, m, "spaced_repeat")
	assert.Contains(t, m, "delay_after_ms")
}

func TestLesson_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	lesson := models.Lesson{
		ID:        "lesson-1",
		CourseID:  "course-1",
		Sequence:  1,
		Title:     "Lesson One",
		CreatedAt: now,
		Turns: []models.Turn{
			{ID: "t1", Sequence: 1, Speaker: models.SpeakerSystem, Text: "Hi"},
		},
	}

	data, err := json.Marshal(lesson)
	require.NoError(t, err)

	var got models.Lesson
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, lesson, got)
}

func TestLesson_JSONFieldNames(t *testing.T) {
	t.Parallel()
	l := models.Lesson{ID: "l1"}
	data, err := json.Marshal(l)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "course_id")
	assert.Contains(t, m, "sequence")
	assert.Contains(t, m, "title")
	assert.Contains(t, m, "turns")
	assert.Contains(t, m, "created_at")
}

func TestCourse_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	course := models.Course{
		ID:          "course-1",
		Title:       "Spanish Basics",
		Description: "Learn Spanish basics",
		SourceLang:  "en",
		TargetLang:  "es",
		Direction:   models.DirectionForward,
		Perspective: models.PerspectiveMale,
		BlueprintID: "travel-basics-v1",
		LessonCount: 3,
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: "admin-1",
		Lessons:     []models.Lesson{},
	}

	data, err := json.Marshal(course)
	require.NoError(t, err)

	var got models.Course
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, course, got)
}

func TestCourse_JSONFieldNames(t *testing.T) {
	t.Parallel()
	c := models.Course{ID: "c1", Direction: models.DirectionReverse}
	data, err := json.Marshal(c)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "title")
	assert.Contains(t, m, "description")
	assert.Contains(t, m, "source_lang")
	assert.Contains(t, m, "target_lang")
	assert.Contains(t, m, "direction")
	assert.Contains(t, m, "perspective")
	assert.Contains(t, m, "blueprint_id")
	assert.Contains(t, m, "lesson_count")
	assert.Contains(t, m, "created_at")
	assert.Contains(t, m, "generated_at")
	assert.Contains(t, m, "generated_by")
	assert.Contains(t, m, "lessons")
}

func TestLessonProgress_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	lp := models.LessonProgress{
		LessonID:    "lesson-1",
		Sequence:    1,
		CompletedAt: now,
	}

	data, err := json.Marshal(lp)
	require.NoError(t, err)

	var got models.LessonProgress
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, lp, got)
}

func TestLessonProgress_JSONFieldNames(t *testing.T) {
	t.Parallel()
	lp := models.LessonProgress{LessonID: "l1"}
	data, err := json.Marshal(lp)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "lesson_id")
	assert.Contains(t, m, "sequence")
	assert.Contains(t, m, "completed_at")
}

func TestCourseProgress_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	cp := models.CourseProgress{
		UserID:         "user-1",
		CourseID:       "course-1",
		CurrentLesson:  2,
		LastAccessedAt: now,
		UpdatedAt:      now,
		LessonsCompleted: []models.LessonProgress{
			{LessonID: "lesson-1", Sequence: 1, CompletedAt: now},
		},
	}

	data, err := json.Marshal(cp)
	require.NoError(t, err)

	var got models.CourseProgress
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, cp, got)
}

func TestCourseProgress_JSONFieldNames(t *testing.T) {
	t.Parallel()
	cp := models.CourseProgress{UserID: "u1", CourseID: "c1"}
	data, err := json.Marshal(cp)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "user_id")
	assert.Contains(t, m, "course_id")
	assert.Contains(t, m, "current_lesson")
	assert.Contains(t, m, "lessons_completed")
	assert.Contains(t, m, "last_accessed_at")
	assert.Contains(t, m, "updated_at")
}

func TestAuditEntry_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)
	entry := models.AuditEntry{
		ID:         "audit-1",
		Timestamp:  now,
		Action:     models.ActionUserCreated,
		ActorID:    "admin-1",
		TargetID:   "user-2",
		TargetType: "user",
		Meta:       map[string]any{"email": "new@example.com"},
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var got models.AuditEntry
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, entry, got)
}

func TestAuditEntry_MetaOmitEmpty(t *testing.T) {
	t.Parallel()
	entry := models.AuditEntry{
		ID:         "audit-2",
		Action:     models.ActionAdminLogin,
		ActorID:    "admin-1",
		TargetType: "session",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	// meta should be omitted when nil
	assert.NotContains(t, m, "meta")
}

func TestAuditEntry_JSONFieldNames(t *testing.T) {
	t.Parallel()
	entry := models.AuditEntry{ID: "a1", Meta: map[string]any{"k": "v"}}
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "timestamp")
	assert.Contains(t, m, "action")
	assert.Contains(t, m, "actor_id")
	assert.Contains(t, m, "target_id")
	assert.Contains(t, m, "target_type")
	assert.Contains(t, m, "meta")
}

func TestScene_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	scene := models.Scene{
		ID:          "scene-1",
		Title:       "At the Airport",
		Description: "Arriving at the airport",
		Vocabulary:  []string{"hello", "goodbye", "thank you"},
	}

	data, err := json.Marshal(scene)
	require.NoError(t, err)

	var got models.Scene
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, scene, got)
}

func TestBlueprint_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	bp := models.Blueprint{
		ID:          "travel-basics-v1",
		Name:        "Travel Basics",
		Description: "Greetings, introductions, asking for help",
		Scenes: []models.Scene{
			{ID: "s1", Title: "Greetings", Vocabulary: []string{"hello", "goodbye"}},
			{ID: "s2", Title: "Introductions", Vocabulary: []string{"my name is", "nice to meet you"}},
		},
	}

	data, err := json.Marshal(bp)
	require.NoError(t, err)

	var got models.Blueprint
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, bp, got)
}

func TestBlueprint_JSONFieldNames(t *testing.T) {
	t.Parallel()
	bp := models.Blueprint{ID: "bp1"}
	data, err := json.Marshal(bp)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "name")
	assert.Contains(t, m, "description")
	assert.Contains(t, m, "scenes")
}
