package models

import "time"

// CourseDirection indicates whether a course goes from native → target or target → native.
type CourseDirection string

const (
	DirectionForward CourseDirection = "forward"
	DirectionReverse CourseDirection = "reverse"
)

// Perspective indicates the gender perspective of the learner in generated content.
type Perspective string

const (
	PerspectiveMale   Perspective = "male"
	PerspectiveFemale Perspective = "female"
)

// TurnSpeaker identifies who is speaking in a lesson turn.
type TurnSpeaker string

const (
	SpeakerSystem TurnSpeaker = "system"
	SpeakerUser   TurnSpeaker = "user"
)

// AuditAction enumerates the events recorded in the audit log.
type AuditAction string

const (
	ActionUserCreated       AuditAction = "user.created"
	ActionUserDeleted       AuditAction = "user.deleted"
	ActionCourseGenerated   AuditAction = "course.generated"
	ActionCourseDeleted     AuditAction = "course.deleted"
	ActionAdminLogin        AuditAction = "admin.login"
	ActionSpeakingEvaluated AuditAction = "speaking.evaluated"
)

// User represents a registered learner or administrator.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"password_hash"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Turn is a single exchange within a lesson — either system instruction or user practice.
type Turn struct {
	ID           string      `json:"id"`
	Sequence     int         `json:"sequence"`
	Speaker      TurnSpeaker `json:"speaker"`
	Text         string      `json:"text"`
	Translation  string      `json:"translation"`
	AudioFile    string      `json:"audio_file"`
	IsBlurred    bool        `json:"is_blurred"`
	SpacedRepeat bool        `json:"spaced_repeat"`
	DelayAfterMs int         `json:"delay_after_ms"`
}

// Lesson is an ordered sequence of turns within a course.
type Lesson struct {
	ID        string    `json:"id"`
	CourseID  string    `json:"course_id"`
	Sequence  int       `json:"sequence"`
	Title     string    `json:"title"`
	Turns     []Turn    `json:"turns"`
	CreatedAt time.Time `json:"created_at"`
}

// Course is stored as a single JSON file with all lessons embedded.
type Course struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	SourceLang  string          `json:"source_lang"`
	TargetLang  string          `json:"target_lang"`
	Direction   CourseDirection `json:"direction"`
	Perspective Perspective     `json:"perspective"`
	BlueprintID string          `json:"blueprint_id"`
	LessonCount int             `json:"lesson_count"`
	CreatedAt   time.Time       `json:"created_at"`
	GeneratedAt time.Time       `json:"generated_at"`
	GeneratedBy string          `json:"generated_by"`
	Lessons     []Lesson        `json:"lessons"`
}

// LessonProgress records a single completed lesson within a course.
type LessonProgress struct {
	LessonID    string    `json:"lesson_id"`
	Sequence    int       `json:"sequence"`
	CompletedAt time.Time `json:"completed_at"`
}

// CourseProgress tracks a user's progress through a course.
type CourseProgress struct {
	UserID           string           `json:"user_id"`
	CourseID         string           `json:"course_id"`
	CurrentLesson    int              `json:"current_lesson"`
	LessonsCompleted []LessonProgress `json:"lessons_completed"`
	LastAccessedAt   time.Time        `json:"last_accessed_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// AuditEntry records an admin action or system event.
type AuditEntry struct {
	ID         string         `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	Action     AuditAction    `json:"action"`
	ActorID    string         `json:"actor_id"`
	TargetID   string         `json:"target_id"`
	TargetType string         `json:"target_type"`
	Meta       map[string]any `json:"meta,omitempty"`
}

// Scene defines a thematic context within a blueprint (e.g. "At the Airport").
type Scene struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Vocabulary  []string `json:"vocabulary"`
}

// Blueprint is a built-in course template that defines scene types and lesson sequencing.
type Blueprint struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Scenes      []Scene `json:"scenes"`
}
