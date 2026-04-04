# AGENTS.md

Instructions for AI agents working on this project.

## Project Overview

lang-learn is a self-hosted Pimsleur-style language learning PWA. Go backend, React frontend, file-based JSON stores, LLM course generation via OpenRouter.

## Tech Stack

- **Backend**: Go 1.24+ (toolchain 1.26.1), chi router, bcrypt, JWT
- **Frontend**: React 19, TypeScript, Vite 8
- **Storage**: File-based JSON (no database)
- **Generation**: OpenRouter API (google/gemini-2.5-flash)
- **Deploy**: Docker (multi-stage, single binary with embedded frontend)

## Repository Structure

```
cmd/server/           → Server entrypoint
internal/
  api/                → HTTP handlers, router, middleware, rate limiting
  auth/               → JWT token creation/validation
  config/             → Env-based config
  generator/          → LLM course generation
  models/             → Domain types
  store/              → File-based JSON stores
  testutil/           → Test helpers (MakeUser, MakeCourse, etc.)
  web/                → go:embed frontend dist
frontend/             → React SPA (Vite + TypeScript)
.github/workflows/    → CI (vet, test, build) + E2E (full server boot)
```

## Key Design Decisions

1. **No public registration.** Only admins can create users.
2. **Username-based auth** (no email). Login uses `{username, password, remember_me}`.
3. **Auto-bootstrap**: Server creates `admin:admin` on first start if no users exist.
4. **File-based storage**: All data is JSON files under `DATA_DIR`. No database.
5. **Uniqueness constraint**: Username must be unique (not email).
6. **Named volumes only**: Docker deployment uses named volumes, no bind mounts.

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

## Code Conventions

- All handlers return JSON envelopes: `{"data": ..., "error": ...}`
- Tests use `testutil.MakeUser()` / `testutil.MakeCourse()` factory functions with functional options.
- Pre-commit hooks enforce: go-fmt, go-vet, golangci-lint.
- Commits use conventional commit format: `feat:`, `fix:`, `refactor:`, `chore:`.

## Auth Architecture

- `POST /api/auth/login` accepts `{username, password, remember_me}`
- `remember_me=true` → returns `access_token` + `refresh_token`
- `remember_me=false` → returns `access_token` only (session-only)
- Frontend stores tokens in `localStorage` (remembered) or `sessionStorage` (not remembered)
- Admin middleware checks `is_admin` claim in JWT

## Store Interface Pattern

All stores implement interfaces in `internal/store/store.go`:
- `UserStorer`: Create, GetByID, GetByUsername, GetByEmail, Update, Delete, List
- `CourseStorer`: Create, GetByID, Update, Delete, List
- `ProgressStorer`: Get, Upsert, ListByUser
- `AuditStorer`: Append, ListByDate

## Environment Variables

- `JWT_SECRET` (required) — JWT signing secret
- `DATA_DIR` (default: `/data`) — persistent storage directory
- `PORT` (default: `8080`) — HTTP port
- `LOG_LEVEL` (default: `info`) — log verbosity
- `OPENROUTER_API_KEY` (optional) — enables LLM course generation

## Common Pitfalls

1. Don't add `.envrc` to git — it contains secrets.
2. Don't add user data (courses, users) to git — they belong in `DATA_DIR`.
3. Frontend dist must exist at `internal/web/dist/` for `go:embed` to work.
4. Rate limiting is per-IP on auth endpoints (20 req/min).
5. The `seed` command was removed — use auto-bootstrap + admin API instead.
