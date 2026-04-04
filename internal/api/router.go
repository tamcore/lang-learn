package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/user/lang-learn/internal/auth"
	"github.com/user/lang-learn/internal/generator"
	"github.com/user/lang-learn/internal/store"
	"github.com/user/lang-learn/internal/web"
)

// RouterConfig holds all dependencies needed to construct the API router.
type RouterConfig struct {
	JWTSecret  string
	Users      store.UserStorer
	Courses    store.CourseStorer
	Progress   store.ProgressStorer
	Audit      store.AuditStorer
	CoursesDir string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BcryptCost int
	Gen        *generator.Generator
}

// NewRouter builds the chi router with all API routes mounted.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	authH := NewAuthHandler(cfg.Users, cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL, cfg.BcryptCost)
	courseH := NewCourseHandler(cfg.Courses, cfg.Progress)
	audioH := NewAudioHandler(cfg.CoursesDir)
	adminH := NewAdminHandler(cfg.Users, cfg.Courses, cfg.Audit)

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		// Public auth endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
			r.Post("/logout", authH.Logout)
		})

		// Authenticated endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTSecret))

			r.Get("/courses", courseH.ListCourses)
			r.Get("/courses/{id}", courseH.GetCourse)
			r.Get("/courses/{id}/lessons/{seq}", courseH.GetLesson)

			r.Get("/progress", courseH.GetProgress)
			r.Get("/progress/{courseID}", courseH.GetCourseProgress)
			r.Put("/progress/{courseID}", courseH.UpsertProgress)

			r.Get("/audio/{courseID}/{filename}", audioH.ServeAudio)
		})

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin(cfg.JWTSecret))

			r.Get("/admin/users", adminH.ListUsers)
			r.Get("/admin/users/{id}", adminH.GetUser)
			r.Patch("/admin/users/{id}", adminH.UpdateUser)
			r.Delete("/admin/users/{id}", adminH.DeleteUser)

			r.Get("/admin/courses", adminH.ListCourses)
			r.Delete("/admin/courses/{id}", adminH.DeleteCourse)

			if cfg.Gen != nil {
				genH := NewGenerateHandler(cfg.Gen)
				r.Post("/admin/courses/generate", genH.Generate)
				r.Get("/admin/courses/generate/{jobID}", genH.GetJobStatus)
			}

			r.Get("/admin/audit", adminH.GetAudit)
		})
	})

	// SPA fallback — serve frontend for all non-API routes
	r.NotFound(web.SPAHandler().ServeHTTP)

	return r
}
