# CoachAppBack

`CoachAppBack` is a Go backend for a coaching marketplace application. It exposes a REST API and WebSocket chat for account authentication, onboarding, profile management, coach discovery, session booking, workout program delivery, and real-time messaging.

The service is built with Fiber, PostgreSQL, `pgx`, and `golang-migrate`. OpenAPI documentation is maintained in-repo and can be served locally in development mode.

## Highlights

- JWT-based authentication for `user` and `coach` roles
- Separate onboarding and profile flows for users and coaches
- Coach discovery with filtering and personalized recommendations
- Session booking, payment-state updates, and lifecycle management
- Real-time chat over WebSocket plus conversation/message APIs
- Workout program upload and secure download links
- Optional Supabase Storage integration for avatars and program files
- Embedded API docs viewers for Swagger UI, ReDoc, and Scalar

## Tech Stack

- Go `1.25`
- [Fiber v2](https://github.com/gofiber/fiber)
- PostgreSQL 16
- `pgx/v5` connection pooling
- `golang-migrate` for schema migrations
- JWT for auth
- Supabase Storage for file uploads when configured

## Project Structure

```text
.
├── cmd/
│   ├── migrate/      # Database migration entrypoint
│   └── server/       # API server entrypoint
├── docs/
│   └── openapi.yaml  # OpenAPI source of truth
├── internal/
│   ├── config/       # Environment loading and feature flags
│   ├── database/     # PostgreSQL connection bootstrap
│   ├── handlers/     # HTTP and WebSocket handlers
│   ├── middleware/   # Auth middleware
│   ├── models/       # Domain models
│   ├── repository/   # PostgreSQL data access
│   ├── routes/       # Route registration and docs serving
│   ├── services/     # Business logic
│   └── websocket/    # Chat hub and client lifecycle
├── migrations/       # SQL schema migrations
├── pkg/utils/        # JWT/password helpers
└── docker-compose.yml
```

## Requirements

- Go `1.25+`
- Docker and Docker Compose, if you want the local PostgreSQL instance
- A PostgreSQL database reachable via `DB_URL`
- Supabase project credentials only if you want file upload/download features

## Quick Start

### 1. Start PostgreSQL

```bash
docker compose up db
```

This starts PostgreSQL 16 with:

- database: `coachapp`
- user: `user`
- password: `password`
- port: `5432`

### 2. Create `.env`

```env
PORT=8080
APP_ENV=development
DB_URL=postgres://user:password@localhost:5432/coachapp?sslmode=disable
JWT_SECRET=change-me
ENABLE_API_DOCS=true
```

### 3. Apply migrations

```bash
go run ./cmd/migrate
```

### 4. Start the API

```bash
go run ./cmd/server
```

The server listens on `http://localhost:8080`.

## Environment Variables

### Required

| Variable | Description |
| --- | --- |
| `JWT_SECRET` | Secret used to sign and validate JWT tokens. |
| `DB_URL` | PostgreSQL connection string. The server exits if it is missing. |

### Optional

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port for the Fiber server. |
| `APP_ENV` | `production` | Environment name. Docs are only served when this resolves to `development`. |
| `ENABLE_API_DOCS` | `false` | Enables `/docs` routes, but only in development mode. |
| `SUPABASE_URL` | empty | Supabase project URL for storage operations. |
| `SUPABASE_BUCKET` | empty | Storage bucket used for avatars and program files. |
| `SUPABASE_SERVICE_KEY` | empty | Service role key used for upload, delete, and signed URL generation. |
| `DEFAULT_USER_EMAIL` | empty | Optional bootstrapped account email. |
| `DEFAULT_USER_PASSWORD` | empty | Password for the bootstrapped default account. |
| `DEFAULT_USER_ROLE` | `user` | Role for `DEFAULT_USER_EMAIL`; must be `user` or `coach`. |
| `DEFAULT_COACH_EMAIL` | empty | Optional bootstrapped coach account email. |
| `DEFAULT_COACH_PASSWORD` | empty | Password for the bootstrapped coach account. |

## Storage Behavior

Supabase Storage is optional, but file features depend on it.

- If storage variables are not configured, avatar upload endpoints return `503`.
- If storage variables are not configured, workout program create/download operations also return `503`.
- Signed program download URLs expire after `3600` seconds.

## Database Migrations

Apply all pending migrations:

```bash
go run ./cmd/migrate
```

Roll back migrations:

```bash
go run ./cmd/migrate down
```

The migration command searches for the `migrations/` directory from the current working directory and executable location, so it works both in local development and compiled contexts.

## Development Commands

Run the API:

```bash
go run ./cmd/server
```

Run tests:

```bash
go test ./...
```

Compile-check the project:

```bash
go build ./...
```

## API Documentation

The OpenAPI source of truth lives at [`docs/openapi.yaml`](docs/openapi.yaml).

Docs routes are only exposed when both of the following are true:

- `APP_ENV=development`
- `ENABLE_API_DOCS=true`

Available local docs endpoints:

- `GET /docs`
- `GET /docs/openapi.yaml`
- `GET /docs/swagger`
- `GET /docs/redoc`
- `GET /docs/scalar`

## API Surface

### Public endpoints

- `GET /health`
- `POST /api/auth/register`
- `POST /api/auth/login`

### Authenticated endpoints

- `GET /api/auth/me`
- `POST /api/v1/users/onboarding`
- `GET /api/v1/users/profile`
- `PUT /api/v1/users/profile`
- `POST /api/v1/users/profile/avatar`
- `POST /api/v1/coaches/onboarding`
- `GET /api/v1/coaches`
- `GET /api/v1/coaches/profile`
- `PUT /api/v1/coaches/profile`
- `POST /api/v1/coaches/profile/avatar`
- `GET /api/v1/coaches/recommended`
- `GET /api/v1/coaches/{id}`
- `POST /api/v1/sessions/book`
- `GET /api/v1/sessions`
- `GET /api/v1/sessions/{id}`
- `PUT /api/v1/sessions/{id}/status`
- `POST /api/v1/sessions/{id}/pay`
- `POST /api/v1/programs`
- `GET /api/v1/programs`
- `GET /api/v1/programs/{id}`
- `GET /api/v1/programs/{id}/download`
- `GET /api/v1/conversations`
- `POST /api/v1/conversations`
- `GET /api/v1/conversations/{id}/messages`
- `GET /api/v1/ws` for WebSocket upgrade

### Role behavior

- `user` accounts can register, complete user onboarding, discover coaches, book/pay for sessions, create conversations, and access their programs.
- `coach` accounts can complete coach onboarding, manage coach profiles, update session status, upload workout programs, and participate in chat.

## Example Requests

Register a user:

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "role": "user"
  }'
```

Log in:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Create user onboarding data:

```bash
curl -X POST http://localhost:8080/api/v1/users/onboarding \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Sam User",
    "age": 29,
    "gender": "male",
    "height_cm": 180,
    "weight_kg": 78,
    "fitness_level": "beginner",
    "goals": ["weight_loss", "mobility"],
    "max_hourly_rate": 60,
    "medical_conditions": "asthma"
  }'
```

Discover coaches:

```bash
curl "http://localhost:8080/api/v1/coaches?specialization=weight_loss&min_rating=4&max_price=80&page=1&limit=10" \
  -H "Authorization: Bearer <TOKEN>"
```

Book a session:

```bash
curl -X POST http://localhost:8080/api/v1/sessions/book \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "coach_id": 42,
    "scheduled_at": "2026-03-05T14:00:00Z",
    "duration_minutes": 60,
    "notes": "Focus on mobility and lower back pain."
  }'
```

List sessions:

```bash
curl "http://localhost:8080/api/v1/sessions?timeframe=upcoming" \
  -H "Authorization: Bearer <TOKEN>"
```

Coach uploads a workout program:

```bash
curl -X POST http://localhost:8080/api/v1/programs \
  -H "Authorization: Bearer <COACH_TOKEN>" \
  -F "user_id=12" \
  -F "session_id=44" \
  -F "title=Week 1 Plan" \
  -F "description=Post-session follow-up plan" \
  -F "file=@/absolute/path/to/program.pdf"
```

Get a signed program download URL:

```bash
curl http://localhost:8080/api/v1/programs/10/download \
  -H "Authorization: Bearer <TOKEN>"
```

Create a conversation:

```bash
curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer <USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "coach_id": 42
  }'
```

Open the chat WebSocket:

```bash
wscat -c "ws://localhost:8080/api/v1/ws?token=<TOKEN>"
```

Send a chat message:

```json
{"type":"message","conversation_id":"7","content":"Can we move the session to Friday?"}
```

## Testing

- Unit tests live beside the implementation in `*_test.go` files.
- Integration coverage exists for session flows in `internal/services/session_service_integration_test.go`.
- Database-backed integration tests require `DB_URL` to point to a migrated PostgreSQL instance and skip automatically when the database is unavailable.

Run the full suite with:

```bash
go test ./...
```

## Operational Notes

- WebSocket auth accepts either `?token=<JWT>` or `Authorization: Bearer <JWT>` during the upgrade request.
- `GET /health` returns `{"status":"ok"}` when the service is healthy.
- Local API docs are intentionally development-only and are not exposed in production mode.

## Notes for Contributors

- Keep request/response contract changes in sync with [`docs/openapi.yaml`](docs/openapi.yaml).
- Follow Conventional Commits for commit messages.
- Prefer `gofmt`-formatted, table-driven tests for new behavior.
