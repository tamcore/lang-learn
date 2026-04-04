# lang-learn

A self-hosted, Pimsleur-style language learning PWA. Courses are generated via LLM (OpenRouter) and served from a single Go binary with an embedded React frontend.

## Quick Start

```bash
# 1. Create .envrc (or export manually)
cat > .envrc <<'EOF'
export JWT_SECRET=$(openssl rand -hex 32)
export DATA_DIR=./data
# Optional: enables course generation
export OPENROUTER_API_KEY=sk-or-...
EOF
source .envrc

# 2. Build and run
make dev
# → Server starts on http://localhost:8080
# → Default admin account: admin / admin (change immediately!)
```

## Prerequisites

- Go 1.24+ (toolchain 1.26.1 auto-downloaded)
- Node.js 22+
- npm

## Running Locally

```bash
# Full build + run
make dev

# Or step by step:
make frontend-build   # Build React frontend
make build            # Build Go server
make run              # Run server
```

## Docker

```bash
make docker-build     # Build image: lang-learn:latest
make docker-up        # Start with docker-compose
make docker-down      # Stop
```

The Docker image is a single self-contained binary (~15MB) with the frontend embedded. Data is persisted via a named volume at `/data`.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | — | Secret for JWT signing (min 32 chars) |
| `DATA_DIR` | No | `/data` | Directory for persistent JSON data |
| `PORT` | No | `8080` | HTTP listen port |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |
| `OPENROUTER_API_KEY` | No | — | Enables LLM course generation |

## Architecture

```
cmd/server/          → Server entrypoint, auto-bootstraps admin user
internal/
  api/               → HTTP handlers, routing, middleware, rate limiting
  auth/              → JWT token management
  config/            → Environment-based configuration
  generator/         → LLM course generation (OpenRouter)
  models/            → Domain types (User, Course, Lesson, Progress)
  store/             → File-based JSON stores
  web/               → Embedded frontend (go:embed)
frontend/            → React + TypeScript + Vite SPA
```

### Auth Flow

- **No public registration.** Admin creates users via the admin panel or API.
- Login: `POST /api/auth/login` with `{username, password, remember_me}`
- `remember_me=true`: returns access + refresh token (localStorage)
- `remember_me=false`: returns access token only (sessionStorage)
- On first start, the server creates a default `admin:admin` account.

### API Routes

**Public (rate-limited):**
- `POST /api/auth/login` — Login
- `POST /api/auth/refresh` — Refresh token
- `POST /api/auth/logout` — Logout

**Authenticated:**
- `GET /api/courses` — List courses
- `GET /api/courses/{id}` — Course details
- `GET /api/courses/{id}/lessons/{seq}` — Lesson content
- `GET /api/progress` — User progress
- `PUT /api/progress/{courseID}` — Update progress
- `GET /api/audio/{courseID}/{filename}` — Audio files

**Admin only:**
- `GET/POST /api/admin/users` — List/create users
- `GET/PATCH/DELETE /api/admin/users/{id}` — User CRUD
- `GET/DELETE /api/admin/courses` — Course management
- `POST /api/admin/courses/generate` — Generate course via LLM
- `GET /api/admin/courses/generate/{jobID}` — Generation status
- `GET /api/admin/audit` — Audit log

## Course Generation

Courses are generated using the Pimsleur method with LLM-powered content. Available blueprints:

- **travel-basics-v1** — Greetings, introductions, asking for help
- **restaurant-v1** — Ordering food and drinks
- **directions-v1** — Asking for and giving directions

To generate a course, use the admin panel or:

```bash
curl -X POST http://localhost:8080/api/admin/courses/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_lang": "en",
    "target_lang": "de",
    "blueprint_id": "travel-basics-v1",
    "lesson_count": 5
  }'
```

## Development

```bash
make test             # Run all Go tests
make test-coverage    # Tests with coverage report
make lint             # go vet
```

## License

Private — all rights reserved.
