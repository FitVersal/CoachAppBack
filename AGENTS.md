# Repository Guidelines

## Project Structure & Module Organization
`cmd/server` starts the Fiber API, and `cmd/migrate` applies or rolls back database migrations. Core application code lives under `internal/`: `handlers/` for HTTP entry points, `services/` for business logic, `repository/` for PostgreSQL access, `models/` for shared domain types, `routes/` for route wiring, and `middleware/` for auth and request concerns. Shared helpers belong in `pkg/utils`. SQL migrations live in `migrations/`, and the OpenAPI spec is maintained in `docs/openapi.yaml`.

## Build, Test, and Development Commands
Use `docker compose up db` to start the local PostgreSQL instance from [`docker-compose.yml`](./docker-compose.yml). Start the API with `go run ./cmd/server`. Apply schema changes with `go run ./cmd/migrate`, or roll back the latest migration with `go run ./cmd/migrate down`. Run the full test suite with `go test ./...`. For a production-style compile check, use `go build ./...`.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs for indentation, `gofmt` formatting, and grouped imports. Keep package names short and lowercase; exported identifiers use `PascalCase`, internal helpers use `camelCase`, and environment variables remain uppercase (for example `JWT_SECRET`, `DB_URL`). Match existing file naming: feature code uses descriptive snake_case files such as `profile_handler.go`, and tests live beside the code in `*_test.go`.

## Testing Guidelines
This repository uses Go’s built-in `testing` package. Prefer table-driven tests where input variations are important, and keep fast unit tests next to the package they cover. Database-backed integration tests already exist in [`internal/services/session_service_integration_test.go`](./internal/services/session_service_integration_test.go); they require `DB_URL` to point at a migrated database and will skip automatically when it is unavailable. Add focused assertions for handler status codes, service errors, and repository edge cases.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commits, for example `feat(chat): add real-time chat with websockets and conversation management`. Keep commit subjects imperative and scoped when useful (`feat(session): ...`, `fix(routes): ...`). Pull requests should include a short problem statement, the main implementation notes, test evidence (`go test ./...` output or equivalent), and any API or migration impact. If request or response contracts change, update `docs/openapi.yaml` in the same PR.
