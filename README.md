# lang-learn

A self-hosted, Pimsleur-style language learning PWA. Courses are generated via LLM (OpenRouter) and served from a single Go binary with an embedded React frontend.

## Features

- **Pimsleur method**: Progressive lesson structure with spaced repetition and recall
- **LLM-generated courses**: Create any language pair using configurable LLM models
- **TTS audio**: Text-to-speech for system turns via OpenRouter streaming (pcm16→WAV)
- **Speaking evaluation**: Record speech, transcribe via OpenRouter audio input, score pronunciation
- **PWA**: Installable, offline-capable with service worker
- **Dark/light theme**: Auto-detects system preference, toggleable
- **Chat-bubble UI**: System and user turns displayed as conversation bubbles
- **Audio sequencer**: Auto-play lesson with sequential audio playback
- **Offline sync**: Progress syncs automatically when back online
- **Download for offline**: Pre-cache lesson audio for offline use
- **Mobile responsive**: Touch-optimized with 768px/480px breakpoints
- **Admin panel**: User management, course generation, audio generation, audit log
- **No public registration**: Only admins can create users

## Quick Start

```bash
# 1. Create .envrc (or export manually)
cat > .envrc <<'EOF'
export JWT_SECRET=$(openssl rand -hex 32)
export DATA_DIR=./data
# Optional: enables course generation
export OPENROUTER_API_KEY=sk-or-...
# Optional: TTS and speaking evaluation
export DEFAULT_TTS_MODEL=openai/gpt-audio-mini
export DEFAULT_WHISPER_MODEL=google/gemini-2.5-flash
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

## Docker

```bash
make docker-build     # Build image: lang-learn:latest
make docker-up        # Start with docker-compose
make docker-down      # Stop
```

The Docker image is a single self-contained binary (~15MB) with the frontend embedded. Data is persisted via a named volume at `/data`. **Named volumes only** — no bind mounts.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | — | Secret for JWT signing (min 32 chars) |
| `DATA_DIR` | No | `/data` | Directory for persistent JSON data |
| `PORT` | No | `8080` | HTTP listen port |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |
| `LOG_FORMAT` | No | `json` | Log format: json, text |
| `OPENROUTER_API_KEY` | No | — | Enables LLM course generation |
| `DEFAULT_LLM_MODEL` | No | `google/gemini-2.5-flash` | LLM model for course generation |
| `DEFAULT_TTS_MODEL` | No | — | TTS model (e.g. `openai/gpt-audio-mini`). Empty = TTS disabled |
| `DEFAULT_WHISPER_MODEL` | No | — | STT model (e.g. `google/gemini-2.5-flash`). Empty = disabled |

## Course Generation

Courses are generated using the Pimsleur method with LLM-powered content. Available blueprints:

- **pimsleur-complete-v1** — Full 10-scene progressive arc (recommended)
- **travel-basics-v1** — Greetings, introductions, asking for help
- **restaurant-v1** — Ordering food and drinks
- **directions-v1** — Asking for and giving directions

Generate a course via the admin panel or API:

```bash
curl -X POST http://localhost:8080/api/admin/courses/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_lang": "en",
    "target_lang": "de",
    "blueprint_id": "pimsleur-complete-v1",
    "lesson_count": 10
  }'
```

## Audio Generation

Generate TTS audio for an existing course (requires `DEFAULT_TTS_MODEL`):

```bash
# Generate audio for a specific course
curl -X POST http://localhost:8080/api/admin/courses/$COURSE_ID/audio \
  -H "Authorization: Bearer $TOKEN"

# Check job progress
curl http://localhost:8080/api/admin/courses/generate/$JOB_ID \
  -H "Authorization: Bearer $TOKEN"
```

Or use the "🔊 Audio" button on the admin Courses tab.

## Development

```bash
make dev              # Build + run locally
make test             # Run all Go tests
make test-coverage    # Tests with coverage report
make lint             # go vet
make frontend-build   # Build React frontend
```

## License

Private — all rights reserved.
