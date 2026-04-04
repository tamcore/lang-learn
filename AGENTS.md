# AGENTS.md

Instructions for AI agents working on this project.

## Project Overview

lang-learn is a self-hosted Pimsleur-style language learning PWA. Go backend, React frontend, file-based JSON stores, LLM course generation via OpenRouter, TTS audio, and Whisper-based speaking evaluation.

## Tech Stack

- **Backend**: Go 1.24+ (toolchain 1.26.1), chi router, bcrypt, JWT
- **Frontend**: React 19, TypeScript, Vite 8, vite-plugin-pwa
- **Storage**: File-based JSON (no database)
- **Generation**: OpenRouter API (configurable model, default google/gemini-2.5-flash)
- **TTS**: OpenRouter streaming chat completions with audio output modality (pcm16 → WAV wrapper)
- **STT**: OpenRouter chat completions with audio input modality (base64 encoded)
- **Deploy**: Docker (multi-stage, single binary with embedded frontend)

## Repository Structure

```
cmd/server/           → Server entrypoint
internal/
  api/                → HTTP handlers, router, middleware, rate limiting
  auth/               → JWT token creation/validation
  config/             → Env-based config
  generator/          → LLM course generation, TTS, Whisper
  models/             → Domain types
  store/              → File-based JSON stores
  testutil/           → Test helpers (MakeUser, MakeCourse, etc.)
  web/                → go:embed frontend dist
frontend/
  src/
    api/              → API client + types
    components/       → Reusable UI (lesson/SpeakingFeedback, lesson/DownloadOffline, admin/)
    context/          → AuthContext, ThemeContext
    db/               → IndexedDB wrapper (idb.ts) for offline
    hooks/            → useMediaRecorder, useOnline
    pages/            → LoginPage, CoursesPage, LessonPage, AdminPage
    styles/           → CSS with dark/light theme + mobile responsive
    sync/             → SyncManager for offline progress sync
.github/workflows/    → CI (vet, test, build) + E2E (full server boot)
```

## Key Design Decisions

1. **No public registration.** Only admins can create users.
2. **Username-based auth** (no email). Login uses `{username, password, remember_me}`.
3. **Auto-bootstrap**: Server creates `admin:admin` on first start if no users exist.
4. **File-based storage**: All data is JSON files under `DATA_DIR`. No database.
5. **Uniqueness constraint**: Username must be unique (not email).
6. **Named volumes only**: Docker deployment uses named volumes, no bind mounts.
7. **PWA**: Service worker with Workbox, offline-capable, installable.
8. **Theme**: Dark/light/auto with `prefers-color-scheme` detection.

## Build & Test

```bash
make dev              # Build + run locally
make test             # Go tests (all packages)
make frontend-build   # Build React frontend
make lint             # go vet
```

### Important Build Notes

- Frontend must be built BEFORE `go vet` or `go build` because `internal/web` uses `go:embed all:dist`.
- In CI, use subshells for `cd frontend && ...` to avoid changing the working directory for subsequent steps.
- Go 1.26.1 with `GOTOOLCHAIN=local` requires `golang:1.26-alpine` in Dockerfile (no auto-download).
- `internal/web` has no test files and must be excluded from coverage runs (covdata tool issue).

### Docker Notes

- Base image MUST be `golang:1.26-alpine` (not `1.24`), because `GOTOOLCHAIN=local` in go.mod prevents auto-download.
- Use multi-stage build: builder stage compiles Go + frontend, final stage is `alpine:3` with the binary.
- Named volumes only — the remote SSH host does NOT support bind mounts.
- `DATA_DIR` defaults to `/data` — mount a named volume there.

## Code Conventions

- All handlers return JSON envelopes: `{"data": ..., "error": ...}`
- Tests use `testutil.MakeUser()` / `testutil.MakeCourse()` factory functions with functional options.
- Pre-commit hooks enforce: go-fmt, go-vet, golangci-lint.
- Commits use conventional commit format: `feat:`, `fix:`, `refactor:`, `chore:`.
- TDD: write tests FIRST, target ≥90% coverage.

## Auth Architecture

- `POST /api/auth/login` accepts `{username, password, remember_me}`
- `remember_me=true` → returns `access_token` + `refresh_token`
- `remember_me=false` → returns `access_token` only (session-only)
- Frontend stores tokens in `localStorage` (remembered) or `sessionStorage` (not remembered)
- Admin middleware checks `is_admin` claim in JWT

## API Endpoints

### Auth
- `POST /api/auth/login` — Login with username + password
- `POST /api/auth/refresh` — Refresh access token
- `POST /api/auth/logout` — Logout

### Courses & Lessons
- `GET /api/courses` — List all courses
- `GET /api/courses/{id}` — Get course details
- `GET /api/courses/{id}/lessons/{seq}` — Get lesson with turns

### Progress
- `GET /api/progress` — Get all progress
- `GET /api/progress/{courseID}` — Get course progress
- `PUT /api/progress/{courseID}` — Update progress

### Audio
- `GET /api/audio/{courseID}/{filename}` — Serve audio files (Range header support)

### Speaking
- `POST /api/speaking/evaluate` — Evaluate speaking (base64 audio → transcription → score)

### Admin
- `GET/POST /api/admin/users` — List/create users
- `GET/PATCH/DELETE /api/admin/users/{id}` — Get/update/delete user
- `GET/DELETE /api/admin/courses` — List/delete courses
- `POST /api/admin/courses/generate` — Generate course via LLM
- `POST /api/admin/courses/{id}/audio` — Generate TTS audio for existing course
- `GET /api/admin/courses/generate/{jobID}` — Check generation job status
- `GET /api/admin/audit` — View audit log

## TTS / STT Implementation

- **TTS** uses OpenRouter `/chat/completions` with `modalities: ["text", "audio"]` and `stream: true`.
  - Format MUST be `pcm16` when streaming (wav not supported with stream=true).
  - Raw PCM16 audio is wrapped in a 44-byte WAV header (24kHz, mono, 16-bit) for browser playback.
  - Response is SSE with `delta.audio.data` containing base64 chunks — collect all chunks, join, base64-decode.
  - Default model: `openai/gpt-audio-mini` ($0.60/$2.40 per M tokens).
  - Concurrency capped at 3 via semaphore.
  - Audio files stored as `.wav` in `data/courses/{id}/audio/{turnID}.wav`.
- **STT** uses OpenRouter `/chat/completions` with `input_audio` content type (NOT the `/audio/transcriptions` endpoint).
  - Sends base64-encoded audio as part of message content.
  - Default model: `google/gemini-2.5-flash`.
- **OpenRouter does NOT support** `/audio/speech` or `/audio/transcriptions` endpoints (returns 404 HTML).
- `tts.go` uses `ttsChatMessage`, `ttsChatRequest`, `ttsAudioConfig` to avoid type conflicts with `llm.go`.

## Store Interface Pattern

All stores implement interfaces in `internal/store/store.go`:
- `UserStorer`: Create, GetByID, GetByUsername, GetByEmail, Update, Delete, List
- `CourseStorer`: Create, GetByID, Update, Delete, List
- `ProgressStorer`: Get, Upsert, ListByUser
- `AuditStorer`: Append, ListByDate

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | (required) | JWT signing secret |
| `DATA_DIR` | `/data` | Persistent storage directory |
| `PORT` | `8080` | HTTP port |
| `LOG_LEVEL` | `info` | Log verbosity (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | Log format (json/text) |
| `OPENROUTER_API_KEY` | (optional) | Enables LLM course generation |
| `DEFAULT_LLM_MODEL` | `google/gemini-2.5-flash` | LLM model for course generation |
| `DEFAULT_TTS_MODEL` | (empty = disabled) | TTS model (e.g. `openai/gpt-audio-mini`) |
| `DEFAULT_WHISPER_MODEL` | (empty = disabled) | STT model (e.g. `google/gemini-2.5-flash`) |

## Common Pitfalls

1. Don't add `.envrc` to git — it contains secrets.
2. Don't add user data (courses, users) to git — they belong in `DATA_DIR`.
3. Frontend dist must exist at `internal/web/dist/` for `go:embed` to work.
4. Rate limiting is per-IP on auth endpoints (20 req/min).
5. The `seed` command was removed — use auto-bootstrap + admin API instead.
6. `go test` coverage: exclude `internal/web` (no test files + covdata issue).
