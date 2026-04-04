// Command seed creates the initial users and triggers course generation.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/user/lang-learn/internal/generator"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		slog.Error("OPENROUTER_API_KEY is required")
		os.Exit(1)
	}

	ctx := context.Background()

	// Init stores
	users, err := store.NewFileUserStore(filepath.Join(dataDir, "users"))
	if err != nil {
		slog.Error("user store", "err", err)
		os.Exit(1)
	}

	courses, err := store.NewFileCourseStore(filepath.Join(dataDir, "courses"))
	if err != nil {
		slog.Error("course store", "err", err)
		os.Exit(1)
	}

	audit, err := store.NewFileAuditStore(filepath.Join(dataDir, "audit"))
	if err != nil {
		slog.Error("audit store", "err", err)
		os.Exit(1)
	}

	// Seed users
	now := time.Now().UTC()

	philippHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
	philipp := models.User{
		ID:           "philipp",
		Username:     "philipp",
		Email:        "philipp@lang-learn.local",
		PasswordHash: string(philippHash),
		IsAdmin:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	jankaHash, _ := bcrypt.GenerateFromPassword([]byte("user123"), 12)
	janka := models.User{
		ID:           "janka",
		Username:     "janka",
		Email:        "janka@lang-learn.local",
		PasswordHash: string(jankaHash),
		IsAdmin:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	for _, u := range []models.User{philipp, janka} {
		if err := users.Create(ctx, u); err != nil {
			slog.Warn("user may already exist", "user", u.Username, "err", err)
		} else {
			slog.Info("created user", "username", u.Username, "admin", u.IsAdmin)
		}
	}

	// Generate course pairs
	llm := generator.NewLLMClient(apiKey, "")
	gen := generator.NewGenerator(llm, courses, audit)

	type coursePair struct {
		source string
		target string
	}

	pairs := []coursePair{
		{source: "sk", target: "en"},
		{source: "en", target: "de"},
	}

	var jobIDs []string

	for _, pair := range pairs {
		for _, dir := range []models.CourseDirection{models.DirectionForward, models.DirectionReverse} {
			src, tgt := pair.source, pair.target
			if dir == models.DirectionReverse {
				src, tgt = tgt, src
			}

			slog.Info("starting course generation", "source", src, "target", tgt, "direction", dir)
			jobID := gen.Generate(generator.GenerateRequest{
				BlueprintID: "travel-basics-v1",
				SourceLang:  src,
				TargetLang:  tgt,
				Direction:   dir,
				Perspective: models.PerspectiveMale,
				LessonCount: 3,
				ActorID:     "philipp",
			})
			jobIDs = append(jobIDs, jobID)
		}
	}

	// Wait for all jobs to complete
	slog.Info("waiting for course generation", "jobs", len(jobIDs))
	for _, jobID := range jobIDs {
		for {
			job, ok := gen.GetJob(jobID)
			if !ok {
				break
			}
			if job.Status == "completed" {
				slog.Info("job completed", "job", jobID, "course", job.CourseID)
				break
			}
			if job.Status == "failed" {
				slog.Error("job failed", "job", jobID, "err", job.Error)
				break
			}
			time.Sleep(2 * time.Second)
			fmt.Printf("  %s: %.0f%% (%s)\n", jobID, job.Progress*100, job.Status)
		}
	}

	slog.Info("seed complete")
}
