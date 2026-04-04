package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/user/lang-learn/internal/api"
	"github.com/user/lang-learn/internal/config"
	"github.com/user/lang-learn/internal/generator"
	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	setupLogger(cfg.LogLevel)

	users, err := store.NewFileUserStore(filepath.Join(cfg.DataDir, "users"))
	if err != nil {
		slog.Error("failed to init user store", "err", err)
		os.Exit(1)
	}

	bootstrapAdmin(users, cfg.BcryptCost)

	courses, err := store.NewFileCourseStore(filepath.Join(cfg.DataDir, "courses"))
	if err != nil {
		slog.Error("failed to init course store", "err", err)
		os.Exit(1)
	}

	progress, err := store.NewFileProgressStore(filepath.Join(cfg.DataDir, "progress"))
	if err != nil {
		slog.Error("failed to init progress store", "err", err)
		os.Exit(1)
	}

	audit, err := store.NewFileAuditStore(filepath.Join(cfg.DataDir, "audit"))
	if err != nil {
		slog.Error("failed to init audit store", "err", err)
		os.Exit(1)
	}

	var gen *generator.Generator
	if cfg.OpenRouterAPIKey != "" {
		llm := generator.NewLLMClient(cfg.OpenRouterAPIKey, "")
		gen = generator.NewGenerator(llm, courses, audit)
	}

	router := api.NewRouter(api.RouterConfig{
		JWTSecret:  cfg.JWTSecret,
		Users:      users,
		Courses:    courses,
		Progress:   progress,
		Audit:      audit,
		CoursesDir: filepath.Join(cfg.DataDir, "courses"),
		AccessTTL:  cfg.AccessTokenTTL,
		RefreshTTL: cfg.RefreshTokenTTL,
		BcryptCost: cfg.BcryptCost,
		Gen:        gen,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}

// bootstrapAdmin creates a default admin user if no users exist.
func bootstrapAdmin(users store.UserStorer, bcryptCost int) {
	existing, err := users.List(context.Background())
	if err != nil {
		slog.Error("failed to list users for bootstrap", "err", err)
		os.Exit(1)
	}
	if len(existing) > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcryptCost)
	if err != nil {
		slog.Error("failed to hash bootstrap password", "err", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	admin := models.User{
		ID:           "admin",
		Username:     "admin",
		PasswordHash: string(hash),
		IsAdmin:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := users.Create(context.Background(), admin); err != nil {
		slog.Error("failed to create bootstrap admin", "err", err)
		os.Exit(1)
	}
	slog.Info("bootstrapped default admin user (admin:admin) — change the password!")
}
