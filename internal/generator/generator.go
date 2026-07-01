package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
)

// JobStatus represents the state of a course generation job.
type JobStatus struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // "pending", "running", "completed", "failed"
	Progress  float64   `json:"progress"`
	CourseID  string    `json:"course_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Generator orchestrates course generation using LLM and TTS.
type Generator struct {
	llm     *LLMClient
	tts     *TTSClient // optional, nil disables audio generation
	courses store.CourseStorer
	audit   store.AuditStorer
	dataDir string // root data directory for saving audio files
	jobs    sync.Map
}

// NewGenerator creates a Generator. tts may be nil to skip audio generation.
func NewGenerator(llm *LLMClient, tts *TTSClient, courses store.CourseStorer, audit store.AuditStorer, dataDir string) *Generator {
	return &Generator{llm: llm, tts: tts, courses: courses, audit: audit, dataDir: dataDir}
}

// GetJob returns the current status of a generation job.
func (g *Generator) GetJob(id string) (JobStatus, bool) {
	v, ok := g.jobs.Load(id)
	if !ok {
		return JobStatus{}, false
	}
	return v.(JobStatus), true
}

// GenerateRequest contains the parameters for course generation.
type GenerateRequest struct {
	BlueprintID string
	SourceLang  string
	TargetLang  string
	Direction   models.CourseDirection
	Perspective models.Perspective
	LessonCount int
	ActorID     string
}

// Generate starts an async course generation job and returns the job ID immediately.
func (g *Generator) Generate(req GenerateRequest) string {
	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())
	g.jobs.Store(jobID, JobStatus{
		ID:        jobID,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	})

	go g.run(jobID, req)
	return jobID
}

func (g *Generator) run(jobID string, req GenerateRequest) {
	g.updateJob(jobID, "running", 0, "", "")
	ctx := context.Background()

	blueprint, ok := Blueprints()[req.BlueprintID]
	if !ok {
		g.updateJob(jobID, "failed", 0, "", "unknown blueprint: "+req.BlueprintID)
		return
	}

	// Step 1: Generate lesson titles
	slog.Info("generating lesson titles", "job", jobID)
	outlinePrompt := BuildLessonOutlinePrompt(blueprint, req.SourceLang, req.TargetLang, req.LessonCount)

	var titles []string
	var err error
	for range 3 {
		resp, llmErr := g.llm.Complete(ctx, outlinePrompt)
		if llmErr != nil {
			err = llmErr
			continue
		}
		resp = cleanJSON(resp)
		if jsonErr := json.Unmarshal([]byte(resp), &titles); jsonErr != nil {
			err = fmt.Errorf("parse titles: %w (raw: %s)", jsonErr, resp)
			continue
		}
		err = nil
		break
	}
	if err != nil {
		g.updateJob(jobID, "failed", 0, "", "lesson titles: "+err.Error())
		return
	}

	g.updateJob(jobID, "running", 0.1, "", "")

	// Step 2: Generate turns for each lesson
	now := time.Now().UTC()
	courseID := fmt.Sprintf("%s-%s-%s-%s", req.SourceLang, req.TargetLang, string(req.Direction), fmt.Sprintf("%d", now.Unix()))

	langNames := map[string]string{
		"sk": "Slovak", "en": "English", "de": "German", "es": "Spanish", "fr": "French",
	}
	srcName := langNames[req.SourceLang]
	if srcName == "" {
		srcName = req.SourceLang
	}
	tgtName := langNames[req.TargetLang]
	if tgtName == "" {
		tgtName = req.TargetLang
	}

	course := models.Course{
		ID:          courseID,
		Title:       fmt.Sprintf("%s → %s (%s)", srcName, tgtName, blueprint.Name),
		Description: fmt.Sprintf("Pimsleur-style %s course: %s to %s, %s perspective", blueprint.Name, srcName, tgtName, req.Perspective),
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Direction:   req.Direction,
		Perspective: req.Perspective,
		BlueprintID: req.BlueprintID,
		LessonCount: len(titles),
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: req.ActorID,
		Lessons:     make([]models.Lesson, 0, len(titles)),
	}

	for i, title := range titles {
		slog.Info("generating lesson", "job", jobID, "lesson", i+1, "title", title)
		progress := 0.1 + (float64(i)/float64(len(titles)))*0.8
		g.updateJob(jobID, "running", progress, "", "")

		turnsPrompt := BuildLessonTurnsPrompt(title, req.SourceLang, req.TargetLang, req.Perspective, i+1)

		var turns []struct {
			Speaker      string `json:"speaker"`
			Text         string `json:"text"`
			Translation  string `json:"translation"`
			IsBlurred    bool   `json:"is_blurred"`
			SpacedRepeat bool   `json:"spaced_repeat"`
			DelayAfterMs int    `json:"delay_after_ms"`
		}

		for range 3 {
			resp, llmErr := g.llm.Complete(ctx, turnsPrompt)
			if llmErr != nil {
				err = llmErr
				continue
			}
			resp = cleanJSON(resp)
			if jsonErr := json.Unmarshal([]byte(resp), &turns); jsonErr != nil {
				err = fmt.Errorf("parse turns for lesson %d: %w", i+1, jsonErr)
				continue
			}
			err = nil
			break
		}

		if err != nil {
			slog.Warn("failed to generate lesson turns, using placeholder", "lesson", i+1, "err", err)
			turns = []struct {
				Speaker      string `json:"speaker"`
				Text         string `json:"text"`
				Translation  string `json:"translation"`
				IsBlurred    bool   `json:"is_blurred"`
				SpacedRepeat bool   `json:"spaced_repeat"`
				DelayAfterMs int    `json:"delay_after_ms"`
			}{
				{Speaker: "system", Text: "Welcome to " + title, Translation: "Welcome", IsBlurred: false, DelayAfterMs: 2000},
			}
			err = nil
		}

		lessonID := fmt.Sprintf("%s-L%d", courseID, i+1)
		lesson := models.Lesson{
			ID:        lessonID,
			CourseID:  courseID,
			Sequence:  i + 1,
			Title:     title,
			CreatedAt: now,
			Turns:     make([]models.Turn, 0, len(turns)),
		}

		for j, t := range turns {
			turnID := fmt.Sprintf("%s-T%d", lessonID, j+1)
			lesson.Turns = append(lesson.Turns, models.Turn{
				ID:           turnID,
				Sequence:     j + 1,
				Speaker:      models.TurnSpeaker(t.Speaker),
				Text:         t.Text,
				Translation:  t.Translation,
				AudioFile:    "",
				IsBlurred:    t.IsBlurred,
				SpacedRepeat: t.SpacedRepeat,
				DelayAfterMs: t.DelayAfterMs,
			})
		}

		course.Lessons = append(course.Lessons, lesson)
	}

	// Step 3: Generate TTS audio for system turns (if TTS client configured)
	if g.tts != nil {
		g.updateJob(jobID, "running", 0.9, "", "")
		audioDir := filepath.Join(g.dataDir, "courses", courseID, "audio")
		if err := os.MkdirAll(audioDir, 0o755); err != nil {
			slog.Warn("failed to create audio dir", "err", err)
		} else {
			for li := range course.Lessons {
				for ti := range course.Lessons[li].Turns {
					turn := &course.Lessons[li].Turns[ti]
					if turn.Speaker != models.SpeakerSystem || turn.Text == "" {
						continue
					}
					filename := fmt.Sprintf("%s.wav", turn.ID)
					data, ttsErr := g.tts.Synthesize(ctx, turn.Text)
					if ttsErr != nil {
						slog.Warn("tts failed for turn", "turn", turn.ID, "err", ttsErr)
						continue
					}
					if writeErr := os.WriteFile(filepath.Join(audioDir, filename), data, 0o644); writeErr != nil {
						slog.Warn("failed to write audio", "turn", turn.ID, "err", writeErr)
						continue
					}
					turn.AudioFile = "audio/" + filename
				}
			}
			slog.Info("tts generation complete", "job", jobID, "course", courseID)
		}
	}

	// Step 4: Save course
	if err := g.courses.Create(ctx, course); err != nil {
		g.updateJob(jobID, "failed", 0, "", "save course: "+err.Error())
		return
	}

	// Step 4: Audit log
	_ = g.audit.Append(ctx, models.AuditEntry{
		ID:         fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Timestamp:  time.Now().UTC(),
		Action:     models.ActionCourseGenerated,
		ActorID:    req.ActorID,
		TargetID:   courseID,
		TargetType: "course",
		Meta:       map[string]any{"blueprint": req.BlueprintID, "source": req.SourceLang, "target": req.TargetLang},
	})

	g.updateJob(jobID, "completed", 1.0, courseID, "")
	slog.Info("course generation complete", "job", jobID, "course", courseID)
}

// GenerateAudio generates TTS audio for all system turns of an existing course.
// Skips turns that already have audio files on disk. Returns job ID for tracking.
func (g *Generator) GenerateAudio(courseID, actorID string) (string, error) {
	if g.tts == nil {
		return "", fmt.Errorf("TTS not configured (set DEFAULT_TTS_MODEL)")
	}

	course, err := g.courses.GetByID(context.Background(), courseID)
	if err != nil {
		return "", fmt.Errorf("course not found: %w", err)
	}

	jobID := fmt.Sprintf("audio-%d", time.Now().UnixNano())
	g.jobs.Store(jobID, JobStatus{
		ID:        jobID,
		Status:    "pending",
		CourseID:  courseID,
		CreatedAt: time.Now().UTC(),
	})

	go g.runAudio(jobID, course, actorID)
	return jobID, nil
}

func (g *Generator) runAudio(jobID string, course models.Course, actorID string) {
	g.updateJob(jobID, "running", 0, course.ID, "")
	ctx := context.Background()

	audioDir := filepath.Join(g.dataDir, "courses", course.ID, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		g.updateJob(jobID, "failed", 0, course.ID, "create audio dir: "+err.Error())
		return
	}

	// Count total system turns to track progress
	var totalTurns, doneTurns int
	for _, lesson := range course.Lessons {
		for _, turn := range lesson.Turns {
			if turn.Speaker == models.SpeakerSystem && turn.Text != "" {
				totalTurns++
			}
		}
	}

	modified := false
	for li := range course.Lessons {
		for ti := range course.Lessons[li].Turns {
			turn := &course.Lessons[li].Turns[ti]
			if turn.Speaker != models.SpeakerSystem || turn.Text == "" {
				continue
			}

			filename := fmt.Sprintf("%s.wav", turn.ID)
			filePath := filepath.Join(audioDir, filename)

			// Skip if audio already exists on disk
			if _, err := os.Stat(filePath); err == nil {
				doneTurns++
				if totalTurns > 0 {
					g.updateJob(jobID, "running", float64(doneTurns)/float64(totalTurns), course.ID, "")
				}
				continue
			}

			slog.Info("generating audio", "job", jobID, "turn", turn.ID, "text", turn.Text[:min(40, len(turn.Text))])
			data, ttsErr := g.tts.Synthesize(ctx, turn.Text)
			if ttsErr != nil {
				slog.Warn("tts failed for turn", "turn", turn.ID, "err", ttsErr)
				doneTurns++
				continue
			}

			if writeErr := os.WriteFile(filePath, data, 0o644); writeErr != nil {
				slog.Warn("failed to write audio", "turn", turn.ID, "err", writeErr)
				doneTurns++
				continue
			}

			turn.AudioFile = course.ID + "/audio/" + filename
			modified = true
			doneTurns++
			if totalTurns > 0 {
				g.updateJob(jobID, "running", float64(doneTurns)/float64(totalTurns), course.ID, "")
			}
		}
	}

	// Update the course JSON if any audio files were added
	if modified {
		if err := g.courses.Update(ctx, course); err != nil {
			slog.Warn("failed to update course with audio paths", "err", err)
		}
	}

	_ = g.audit.Append(ctx, models.AuditEntry{
		ID:         fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Timestamp:  time.Now().UTC(),
		Action:     "audio_generated",
		ActorID:    actorID,
		TargetID:   course.ID,
		TargetType: "course",
		Meta:       map[string]any{"total_turns": totalTurns, "generated": doneTurns},
	})

	g.updateJob(jobID, "completed", 1.0, course.ID, "")
	slog.Info("audio generation complete", "job", jobID, "course", course.ID, "turns", doneTurns)
}

func (g *Generator) updateJob(id, status string, progress float64, courseID, errMsg string) {
	v, ok := g.jobs.Load(id)
	if !ok {
		return
	}
	job := v.(JobStatus)
	job.Status = status
	job.Progress = progress
	job.CourseID = courseID
	job.Error = errMsg
	g.jobs.Store(id, job)
}

// cleanJSON strips markdown fences and leading/trailing whitespace from LLM output.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
