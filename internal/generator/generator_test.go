package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/models"
)

// ---------------------------------------------------------------------------
// Mock stores
// ---------------------------------------------------------------------------

type mockCourseStore struct {
	mu      sync.Mutex
	courses map[string]models.Course
	failOn  string // if set, Create returns an error when called
}

func newMockCourseStore() *mockCourseStore {
	return &mockCourseStore{courses: make(map[string]models.Course)}
}

func (m *mockCourseStore) Create(_ context.Context, c models.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn == "create" {
		return fmt.Errorf("mock: create failed")
	}
	m.courses[c.ID] = c
	return nil
}

func (m *mockCourseStore) GetByID(_ context.Context, id string) (models.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.courses[id]
	if !ok {
		return models.Course{}, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *mockCourseStore) Update(_ context.Context, c models.Course) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.courses[c.ID] = c
	return nil
}

func (m *mockCourseStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.courses, id)
	return nil
}

func (m *mockCourseStore) List(_ context.Context) ([]models.Course, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []models.Course
	for _, c := range m.courses {
		out = append(out, c)
	}
	return out, nil
}

type mockAuditStore struct {
	mu      sync.Mutex
	entries []models.AuditEntry
}

func newMockAuditStore() *mockAuditStore {
	return &mockAuditStore{}
}

func (m *mockAuditStore) Append(_ context.Context, e models.AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
	return nil
}

func (m *mockAuditStore) ListByDate(_ context.Context, _ time.Time) ([]models.AuditEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries, nil
}

// ---------------------------------------------------------------------------
// LLM mock helpers
// ---------------------------------------------------------------------------

// llmChatResponse builds a JSON-encoded chat completion response with the given content.
func llmChatResponse(content string) []byte {
	resp := chatResponse{
		Choices: []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}{
			{Message: struct {
				Content string `json:"content"`
			}{Content: content}},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// newMockLLMServer returns an httptest.Server that handles outline and turn
// prompts. The outline prompt (detected by "lesson titles") returns titlesJSON;
// turn prompts return turnsJSON.
func newMockLLMServer(titlesJSON, turnsJSON string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		prompt := req.Messages[0].Content
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(prompt, "lesson titles") {
			_, _ = w.Write(llmChatResponse(titlesJSON))
		} else {
			_, _ = w.Write(llmChatResponse(turnsJSON))
		}
	}))
}

// ---------------------------------------------------------------------------
// Original tests (prompts, templates, cleanJSON)
// ---------------------------------------------------------------------------

func TestCleanJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", `["a","b"]`, `["a","b"]`},
		{"with fences", "```json\n[\"a\"]\n```", `["a"]`},
		{"with backticks only", "```\n{\"x\":1}\n```", `{"x":1}`},
		{"whitespace", "  [1,2]  ", `[1,2]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cleanJSON(tt.input))
		})
	}
}

func TestBuildLessonOutlinePrompt(t *testing.T) {
	t.Parallel()
	bp := Blueprints()["travel-basics-v1"]
	prompt := BuildLessonOutlinePrompt(bp, "sk", "en", 3)
	assert.Contains(t, prompt, "sk")
	assert.Contains(t, prompt, "en")
	assert.Contains(t, prompt, "3 lesson titles")
}

func TestBuildLessonTurnsPrompt(t *testing.T) {
	t.Parallel()
	prompt := BuildLessonTurnsPrompt("Greetings", "sk", "en", "male", 1)
	assert.Contains(t, prompt, "Greetings")
	assert.Contains(t, prompt, "sk")
	assert.Contains(t, prompt, "male")
}

func TestBuildLessonTurnsPrompt_RecallInstruction(t *testing.T) {
	t.Parallel()
	prompt := BuildLessonTurnsPrompt("Review", "en", "de", "female", 3)
	assert.Contains(t, prompt, "lesson 3")
	assert.Contains(t, prompt, "recall")
}

func TestBlueprintsContainsExpectedKeys(t *testing.T) {
	t.Parallel()
	bp := Blueprints()
	assert.Contains(t, bp, "travel-basics-v1")
	assert.Contains(t, bp, "restaurant-v1")
	assert.Contains(t, bp, "directions-v1")
	assert.Contains(t, bp, "pimsleur-complete-v1")
}

// ---------------------------------------------------------------------------
// NewGenerator tests
// ---------------------------------------------------------------------------

func TestNewGenerator(t *testing.T) {
	t.Parallel()
	llm := NewLLMClient("k", "m")
	tts := NewTTSClient("k", "", "", "")
	cs := newMockCourseStore()
	as := newMockAuditStore()

	g := NewGenerator(llm, tts, cs, as, "/data")
	assert.NotNil(t, g)
	assert.Equal(t, llm, g.llm)
	assert.Equal(t, tts, g.tts)
	assert.Equal(t, "/data", g.dataDir)
}

func TestNewGenerator_NilTTS(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "/data")
	assert.Nil(t, g.tts)
}

// ---------------------------------------------------------------------------
// GetJob tests
// ---------------------------------------------------------------------------

func TestGetJob_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	_, ok := g.GetJob("nonexistent")
	assert.False(t, ok)
}

func TestGetJob_Found(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	g.jobs.Store("test-job", JobStatus{
		ID:     "test-job",
		Status: "running",
	})

	job, ok := g.GetJob("test-job")
	require.True(t, ok)
	assert.Equal(t, "test-job", job.ID)
	assert.Equal(t, "running", job.Status)
}

// ---------------------------------------------------------------------------
// updateJob tests
// ---------------------------------------------------------------------------

func TestUpdateJob(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	g.jobs.Store("j1", JobStatus{ID: "j1", Status: "pending", CreatedAt: time.Now()})

	g.updateJob("j1", "running", 0.5, "", "")
	job, ok := g.GetJob("j1")
	require.True(t, ok)
	assert.Equal(t, "running", job.Status)
	assert.Equal(t, 0.5, job.Progress)
}

func TestUpdateJob_Completed(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	g.jobs.Store("j2", JobStatus{ID: "j2", Status: "running"})

	g.updateJob("j2", "completed", 1.0, "course-123", "")
	job, ok := g.GetJob("j2")
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)
	assert.Equal(t, "course-123", job.CourseID)
	assert.Equal(t, 1.0, job.Progress)
}

func TestUpdateJob_Failed(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	g.jobs.Store("j3", JobStatus{ID: "j3", Status: "running"})

	g.updateJob("j3", "failed", 0, "", "something broke")
	job, ok := g.GetJob("j3")
	require.True(t, ok)
	assert.Equal(t, "failed", job.Status)
	assert.Equal(t, "something broke", job.Error)
}

func TestUpdateJob_NonexistentJob(t *testing.T) {
	t.Parallel()
	g := NewGenerator(NewLLMClient("k", ""), nil, newMockCourseStore(), newMockAuditStore(), "")
	// Should not panic
	g.updateJob("nope", "running", 0, "", "")
	_, ok := g.GetJob("nope")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Generate tests
// ---------------------------------------------------------------------------

func TestGenerate_ReturnsJobID(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1: Greetings"]`
	turnsJSON := `[{"speaker":"system","text":"Hello","translation":"Hola","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL

	g := NewGenerator(llm, nil, newMockCourseStore(), newMockAuditStore(), t.TempDir())
	jobID := g.Generate(GenerateRequest{
		BlueprintID: "travel-basics-v1",
		SourceLang:  "en",
		TargetLang:  "es",
		Direction:   "forward",
		Perspective: "male",
		LessonCount: 1,
		ActorID:     "user-1",
	})

	assert.True(t, strings.HasPrefix(jobID, "job-"))
	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Contains(t, []string{"pending", "running", "completed"}, job.Status)
}

// ---------------------------------------------------------------------------
// run (full pipeline) tests
// ---------------------------------------------------------------------------

func TestRun_FullPipeline_NoTTS(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1: Greetings"]`
	turnsJSON := `[
		{"speaker":"system","text":"Hello","translation":"Hola","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000},
		{"speaker":"user","text":"Hola","translation":"Hello","is_blurred":true,"spaced_repeat":true,"delay_after_ms":5000}
	]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()
	as := newMockAuditStore()

	g := NewGenerator(llm, nil, cs, as, t.TempDir())
	jobID := "test-pipeline-1"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "sk",
		Direction:   "forward",
		Perspective: "male",
		LessonCount: 1,
		ActorID:     "actor-1",
	})

	// Job should be completed
	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)
	assert.Equal(t, 1.0, job.Progress)
	assert.NotEmpty(t, job.CourseID)

	// Course should be stored
	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.Len(t, cs.courses, 1)
	for _, course := range cs.courses {
		assert.Equal(t, "en", course.SourceLang)
		assert.Equal(t, "sk", course.TargetLang)
		assert.Equal(t, "directions-v1", course.BlueprintID)
		require.Len(t, course.Lessons, 1)
		assert.Equal(t, "Lesson 1: Greetings", course.Lessons[0].Title)
		require.Len(t, course.Lessons[0].Turns, 2)
		assert.Equal(t, models.SpeakerSystem, course.Lessons[0].Turns[0].Speaker)
		assert.Equal(t, models.SpeakerUser, course.Lessons[0].Turns[1].Speaker)
		assert.Equal(t, "Hello", course.Lessons[0].Turns[0].Text)
		assert.True(t, course.Lessons[0].Turns[1].IsBlurred)
	}

	// Audit entry should exist
	as.mu.Lock()
	defer as.mu.Unlock()
	require.Len(t, as.entries, 1)
	assert.Equal(t, models.ActionCourseGenerated, as.entries[0].Action)
	assert.Equal(t, "actor-1", as.entries[0].ActorID)
}

func TestRun_FullPipeline_WithTTS(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1: Hello"]`
	turnsJSON := `[{"speaker":"system","text":"Ahoj","translation":"Hello","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`

	var llmCalls atomic.Int32
	llmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llmCalls.Add(1)
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		prompt := req.Messages[0].Content
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(prompt, "lesson titles") {
			_, _ = w.Write(llmChatResponse(titlesJSON))
		} else {
			_, _ = w.Write(llmChatResponse(turnsJSON))
		}
	}))
	defer llmSrv.Close()

	ttsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseAudioResponse(w, []byte("fake-wav-audio"))
	}))
	defer ttsSrv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = llmSrv.URL
	tts := NewTTSClient("key", "", "", ttsSrv.URL)
	cs := newMockCourseStore()
	as := newMockAuditStore()
	dataDir := t.TempDir()

	g := NewGenerator(llm, tts, cs, as, dataDir)
	jobID := "test-tts-pipeline"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "sk",
		Direction:   "forward",
		Perspective: "female",
		LessonCount: 1,
		ActorID:     "actor-2",
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)

	// Verify audio file was created
	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.Len(t, cs.courses, 1)
	for _, course := range cs.courses {
		require.Len(t, course.Lessons, 1)
		turn := course.Lessons[0].Turns[0]
		assert.NotEmpty(t, turn.AudioFile, "system turn should have audio file set")
		assert.Contains(t, turn.AudioFile, "audio/")

		// Check the actual file exists on disk
		audioPath := filepath.Join(dataDir, "courses", course.ID, turn.AudioFile)
		data, err := os.ReadFile(audioPath)
		require.NoError(t, err)
		assert.Equal(t, "RIFF", string(data[:4]))
		assert.Equal(t, []byte("fake-wav-audio"), data[44:])
	}
}

func TestRun_UnknownBlueprint(t *testing.T) {
	t.Parallel()
	llm := NewLLMClient("key", "model")
	g := NewGenerator(llm, nil, newMockCourseStore(), newMockAuditStore(), t.TempDir())

	jobID := "test-bad-bp"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "nonexistent-blueprint",
		SourceLang:  "en",
		TargetLang:  "de",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "failed", job.Status)
	assert.Contains(t, job.Error, "unknown blueprint")
}

func TestRun_LLMFailsOnOutline(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	g := NewGenerator(llm, nil, newMockCourseStore(), newMockAuditStore(), t.TempDir())

	jobID := "test-llm-fail"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "de",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "failed", job.Status)
	assert.Contains(t, job.Error, "lesson titles")
}

func TestRun_LLMReturnsInvalidOutlineJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(llmChatResponse("not a json array"))
	}))
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	g := NewGenerator(llm, nil, newMockCourseStore(), newMockAuditStore(), t.TempDir())

	jobID := "test-bad-json"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "fr",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "failed", job.Status)
	assert.Contains(t, job.Error, "lesson titles")
}

func TestRun_LLMReturnsInvalidTurnsJSON_FallsBackToPlaceholder(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		prompt := req.Messages[0].Content
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(prompt, "lesson titles") {
			_, _ = w.Write(llmChatResponse(`["Greetings"]`))
		} else {
			// Return invalid turns JSON — should trigger placeholder fallback
			_, _ = w.Write(llmChatResponse(`not valid json for turns`))
		}
	}))
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()

	g := NewGenerator(llm, nil, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-turns-fallback"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "de",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status, "should complete even with placeholder turns")

	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.Len(t, cs.courses, 1)
	for _, course := range cs.courses {
		require.Len(t, course.Lessons, 1)
		require.Len(t, course.Lessons[0].Turns, 1)
		assert.Equal(t, models.TurnSpeaker("system"), course.Lessons[0].Turns[0].Speaker)
		assert.Contains(t, course.Lessons[0].Turns[0].Text, "Welcome to")
	}
}

func TestRun_CourseStoreFails(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1"]`
	turnsJSON := `[{"speaker":"system","text":"Hi","translation":"Hi","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()
	cs.failOn = "create"

	g := NewGenerator(llm, nil, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-store-fail"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "es",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "failed", job.Status)
	assert.Contains(t, job.Error, "save course")
}

func TestRun_MultipleLesson(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1: Basics", "Lesson 2: Numbers"]`
	turnsJSON := `[{"speaker":"system","text":"Test","translation":"Test","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()

	g := NewGenerator(llm, nil, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-multi-lesson"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "restaurant-v1",
		SourceLang:  "de",
		TargetLang:  "en",
		Direction:   "reverse",
		Perspective: "female",
		LessonCount: 2,
		ActorID:     "user-2",
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)

	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.Len(t, cs.courses, 1)
	for _, course := range cs.courses {
		assert.Equal(t, 2, course.LessonCount)
		require.Len(t, course.Lessons, 2)
		assert.Equal(t, 1, course.Lessons[0].Sequence)
		assert.Equal(t, 2, course.Lessons[1].Sequence)
		assert.Equal(t, "Lesson 1: Basics", course.Lessons[0].Title)
		assert.Equal(t, "Lesson 2: Numbers", course.Lessons[1].Title)
	}
}

func TestRun_LangNameFallback(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1"]`
	turnsJSON := `[{"speaker":"system","text":"Hi","translation":"Hi","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()

	g := NewGenerator(llm, nil, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-lang-fallback"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "pt",
		TargetLang:  "ja",
		Direction:   "forward",
		Perspective: "male",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)

	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, course := range cs.courses {
		// Unknown lang codes should fall back to the raw code
		assert.Contains(t, course.Title, "pt")
		assert.Contains(t, course.Title, "ja")
	}
}

func TestRun_TTSFailsGracefully(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1"]`
	turnsJSON := `[{"speaker":"system","text":"Ahoj","translation":"Hello","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	llmSrv := newMockLLMServer(titlesJSON, turnsJSON)
	defer llmSrv.Close()

	// TTS server that always fails
	ttsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`tts error`))
	}))
	defer ttsSrv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = llmSrv.URL
	tts := NewTTSClient("key", "", "", ttsSrv.URL)
	cs := newMockCourseStore()

	g := NewGenerator(llm, tts, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-tts-fail"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "sk",
		LessonCount: 1,
	})

	// Course should still be created even if TTS fails
	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)

	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, course := range cs.courses {
		// Audio file should be empty since TTS failed
		assert.Empty(t, course.Lessons[0].Turns[0].AudioFile)
	}
}

func TestRun_TTSSkipsUserTurns(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Lesson 1"]`
	turnsJSON := `[
		{"speaker":"system","text":"Hello","translation":"Hola","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000},
		{"speaker":"user","text":"Hola","translation":"Hello","is_blurred":true,"spaced_repeat":true,"delay_after_ms":5000}
	]`
	llmSrv := newMockLLMServer(titlesJSON, turnsJSON)
	defer llmSrv.Close()

	var ttsCalls atomic.Int32
	ttsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ttsCalls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio"))
	}))
	defer ttsSrv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = llmSrv.URL
	tts := NewTTSClient("key", "", "", ttsSrv.URL)
	cs := newMockCourseStore()

	g := NewGenerator(llm, tts, cs, newMockAuditStore(), t.TempDir())
	jobID := "test-tts-user"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "es",
		LessonCount: 1,
	})

	job, ok := g.GetJob(jobID)
	require.True(t, ok)
	assert.Equal(t, "completed", job.Status)
	// Only 1 TTS call for the system turn, not the user turn
	assert.Equal(t, int32(1), ttsCalls.Load())
}

// ---------------------------------------------------------------------------
// Generate async integration test
// ---------------------------------------------------------------------------

func TestGenerate_AsyncCompletesSuccessfully(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Async Lesson"]`
	turnsJSON := `[{"speaker":"system","text":"Test","translation":"Test","is_blurred":false,"spaced_repeat":false,"delay_after_ms":1000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()

	g := NewGenerator(llm, nil, cs, newMockAuditStore(), t.TempDir())
	jobID := g.Generate(GenerateRequest{
		BlueprintID: "directions-v1",
		SourceLang:  "en",
		TargetLang:  "de",
		Direction:   "forward",
		Perspective: "male",
		LessonCount: 1,
		ActorID:     "tester",
	})

	// Poll until completed (with timeout)
	deadline := time.Now().Add(10 * time.Second)
	var job JobStatus
	for time.Now().Before(deadline) {
		var ok bool
		job, ok = g.GetJob(jobID)
		if ok && (job.Status == "completed" || job.Status == "failed") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.Equal(t, "completed", job.Status)
	assert.NotEmpty(t, job.CourseID)

	cs.mu.Lock()
	defer cs.mu.Unlock()
	assert.Len(t, cs.courses, 1)
}

// ---------------------------------------------------------------------------
// Course metadata tests
// ---------------------------------------------------------------------------

func TestRun_CourseMetadata(t *testing.T) {
	t.Parallel()
	titlesJSON := `["Basics"]`
	turnsJSON := `[{"speaker":"system","text":"Hi","translation":"Hi","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000}]`
	srv := newMockLLMServer(titlesJSON, turnsJSON)
	defer srv.Close()

	llm := NewLLMClient("key", "model")
	llm.baseURL = srv.URL
	cs := newMockCourseStore()
	as := newMockAuditStore()

	g := NewGenerator(llm, nil, cs, as, t.TempDir())
	jobID := "test-metadata"
	g.jobs.Store(jobID, JobStatus{ID: jobID, Status: "pending", CreatedAt: time.Now()})

	g.run(jobID, GenerateRequest{
		BlueprintID: "travel-basics-v1",
		SourceLang:  "en",
		TargetLang:  "de",
		Direction:   "forward",
		Perspective: "female",
		LessonCount: 1,
		ActorID:     "admin",
	})

	cs.mu.Lock()
	defer cs.mu.Unlock()
	require.Len(t, cs.courses, 1)
	for _, course := range cs.courses {
		assert.Equal(t, "en", course.SourceLang)
		assert.Equal(t, "de", course.TargetLang)
		assert.Equal(t, models.CourseDirection("forward"), course.Direction)
		assert.Equal(t, models.Perspective("female"), course.Perspective)
		assert.Equal(t, "travel-basics-v1", course.BlueprintID)
		assert.Contains(t, course.Title, "English")
		assert.Contains(t, course.Title, "German")
		assert.Contains(t, course.Title, "Travel Basics")
		assert.Contains(t, course.Description, "female")
		assert.Equal(t, "admin", course.GeneratedBy)
		assert.False(t, course.CreatedAt.IsZero())
		assert.False(t, course.GeneratedAt.IsZero())

		// Lesson IDs should contain the course ID
		assert.Contains(t, course.Lessons[0].ID, course.ID)
		assert.Equal(t, course.ID, course.Lessons[0].CourseID)
	}

	// Audit
	as.mu.Lock()
	defer as.mu.Unlock()
	require.Len(t, as.entries, 1)
	e := as.entries[0]
	assert.Equal(t, "admin", e.ActorID)
	assert.Equal(t, "course", e.TargetType)
	assert.Equal(t, "travel-basics-v1", e.Meta["blueprint"])
}
