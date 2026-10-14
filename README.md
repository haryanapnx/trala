# Task Management API (Multi-User)

REST API for managing per-user tasks, built in **Go** with **clean architecture**.
Each user owns their own tasks, can assign a task to a teammate, and has a
customizable set of statuses/stages.

## Features

- **Authentication** — register & login, JWT (HS256) for protected endpoints.
- **Tasks CRUD** — create, list (status filter, title search, pagination), detail, update, soft delete.
- **Custom statuses** — every user starts with 3 default stages (`todo`, `in_progress`, `done`) and can add, rename, reorder, and delete steps.
- **Assignment** — assign a task to yourself or a teammate in a **single transaction** (update assignee + write `task_logs` + send notification; if any step fails the whole thing rolls back).
- **Idempotency** — `POST /tasks` requires an `Idempotency-Key` header (UUID); a retry with the same key returns an identical response without creating a new task (24h window). Replays are flagged with `X-Idempotent-Replayed: true`; a duplicate that is still being processed is answered with `409 REQUEST_IN_PROGRESS` + `Retry-After`.
- **Structured error handling** — every error uses the JSON shape `{status, code, message, timestamp}`; panics are caught by middleware.
- **Structured logging** — one JSON log line per request with `request_id`, `method`, `path`, `status`, `latency_ms`; level is INFO/WARN/ERROR based on the status code.
- **Swagger UI** — interactive docs at `/docs`.

## Tech Stack

Go 1.21 · [chi](https://github.com/go-chi/chi) (router) · [pgx/v5](https://github.com/jackc/pgx) +
[sqlc](https://sqlc.dev) (queries) · [golang-migrate](https://github.com/golang-migrate/migrate)
(migrations) · [golang-jwt](https://github.com/golang-jwt/jwt) · bcrypt ·
[testify](https://github.com/stretchr/testify) · [swaggo](https://github.com/swaggo/swag) +
[swgui](https://github.com/swaggest/swgui) · [air](https://github.com/air-verse/air) (hot reload).

## Getting Started

Prerequisites: Go 1.21+, Docker, the `migrate` & `sqlc` CLIs (for the related
Makefile targets), and `air` for hot reload (`make install-air`).

```bash
# 1. Prepare configuration
cp .env.example .env

# 2. Start Postgres (the `taskdb` database is created automatically by the container)
make docker-up          # = docker compose up -d db

# 3. Apply schema + dummy data
make migrate-up
make seed               # optional: 3 users, 1 team, 2 sample tasks

# 4. Run the API
make dev                # hot reload via air, listening on :8080
# or: make run
```

Open the API docs at **http://localhost:8080/docs**.

Other Makefile targets: `make build`, `make test`, `make test-cover`, `make lint`,
`make sqlc`, `make swag`, `make migrate-down`, `make migrate-create name=...`,
`make idempotency-prune` (delete expired keys), `make docker-down`.

### Demo credentials (from `make seed`)

| Email | Password |
|---|---|
| alice@example.com | password123 |
| bob@example.com   | password123 |
| carol@example.com | password123 |

All three are members of the "Engineering" team (`aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`).

## Configuration (environment)

See `.env.example`. `DATABASE_URL` and `JWT_SECRET` are **required**.

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP server port |
| `APP_ENV` | `development` | Environment label |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `DATABASE_URL` | — | **required**, Postgres DSN |
| `JWT_SECRET` | — | **required**, JWT signing secret |
| `JWT_EXPIRY` | `24h` | Token lifetime |
| `DB_MAX_CONNS` | `10` | Maximum pool size |
| `DB_MIN_CONNS` | `1` | Minimum idle connections |
| `DB_MAX_CONN_LIFETIME` | `1h` | Maximum connection age |
| `DB_MAX_CONN_IDLE_TIME` | `30m` | Maximum idle time |
| `DB_LOCK_TIMEOUT` | `2s` | Per-transaction row-lock wait limit (in-flight duplicate → `409`) |

## Architecture

Layered clean architecture; dependencies **always point inward**:

```
transport (HTTP)  →  usecase  →  domain  ←  repository (postgres)
```

- `domain` — entities + interfaces (ports) + typed errors. Imports no framework or driver.
- `usecase` — business logic and the **owner of transaction boundaries**; knows nothing about chi/pgx.
- `repository/postgres` — implements the domain ports using sqlc/pgx; the only place SQL lives.
- `transport/http` — chi handlers, middleware, DTOs. Never touches `pgx`.
- `platform` — cross-cutting adapters (jwt, bcrypt, logger, notifier).
- `cmd/api` — composition root: wiring + graceful shutdown.

```
cmd/api/                     # entrypoint
internal/
  config/                    # env loading
  domain/                    # entities, ports, errors
  model/                     # use-case input/output types
  usecase/                   # business logic + transactions
  repository/postgres/       # repositories + sqlc (generated)
  transport/http/            # router, middleware, handlers, dto
  platform/                  # jwt, password, logger, notifier
db/migrations/               # schema (golang-migrate)
db/queries/                  # sqlc query sources
db/seed/                     # dummy data
docs/swagger/                # OpenAPI spec (generated)
```

### Database

Seven tables: `users`, `teams`, `team_members`, `task_statuses`, `tasks`, `task_logs`,
`idempotency_keys`. A few notable choices:

- Case-insensitive login via `UNIQUE (lower(email))`.
- `tasks` uses **soft delete** (`deleted_at`); every read path relies on a partial index `WHERE deleted_at IS NULL`.
- Title search uses a **GIN `pg_trgm`** index.
- `UNIQUE (idempotency_keys.key)` is the anti-duplicate guarantee for concurrent requests.
- `tasks.status_id` → `task_statuses` with `ON DELETE RESTRICT` (a status in use cannot be deleted).

## API

Base path: **`/v1`** (canonical). Health probes are not versioned.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/healthz` | – | Liveness |
| GET | `/readyz` | – | Readiness (pings the DB) |
| GET | `/docs` | – | Swagger UI |
| POST | `/v1/auth/register` | – | Register a user |
| POST | `/v1/auth/login` | – | Log in → JWT |
| GET | `/v1/statuses` | Bearer | List the user's statuses |
| POST | `/v1/statuses` | Bearer | Add a step |
| PUT | `/v1/statuses/{id}` | Bearer | Rename / reorder a step |
| DELETE | `/v1/statuses/{id}` | Bearer | Delete a step |
| POST | `/v1/tasks` | Bearer | Create a task (**requires** the `Idempotency-Key` header) |
| GET | `/v1/tasks` | Bearer | List (`?status=&search=&page=&limit=`) |
| GET | `/v1/tasks/{id}` | Bearer | Detail |
| PUT | `/v1/tasks/{id}` | Bearer | Update |
| DELETE | `/v1/tasks/{id}` | Bearer | Soft delete |
| POST | `/v1/tasks/{id}/assign` | Bearer | Assign to a teammate |

Protected endpoints require an `Authorization: Bearer <token>` header.

Consistent error format:

```json
{ "status": 400, "code": "VALIDATION_ERROR", "message": "title is required", "timestamp": "2026-01-01T00:00:00Z" }
```

### Examples

```bash
# Register + login
curl -X POST localhost:8080/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"dave@example.com","password":"password123","name":"Dave"}'

TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}' | jq -r .access_token)

TEAM=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa

# Create a task (idempotent)
curl -X POST localhost:8080/v1/tasks -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" -H 'Content-Type: application/json' \
  -d "{\"team_id\":\"$TEAM\",\"title\":\"Ship auth\"}"

# List + filter + search + pagination
curl "localhost:8080/v1/tasks?status=todo&search=ship&page=1&limit=10" -H "Authorization: Bearer $TOKEN"

# Assign to a teammate
curl -X POST localhost:8080/v1/tasks/<TASK_ID>/assign -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"assignee_id":"22222222-2222-2222-2222-222222222222"}'
```

## Testing

```bash
make test          # go test ./... -race -count=1
make test-cover    # + coverage report
```

Unit tests run **without a database or any external service** (in-memory fakes). Main focus:

- **Idempotency**: sequential requests with the same key → one task and an identical response; same key with a different payload → `409`.
- **Concurrency**: N goroutines with the same key → exactly one task is created (`go test -race`).

## Design Decisions (summary)

- **Idempotency lives in `TaskUsecase`, not a generic middleware** — it must share the transaction with the task insert.
- **Transactions** are owned by the use case through the `TxManager` port (implementation uses `sqlc.WithTx`); handlers never see `pgx.Tx`.
- **Assign** performs the assignee update + audit log + notification in one transaction (full rollback on failure).
- **Errors** are domain-level (no HTTP status) and mapped to HTTP statuses in a single place in the transport layer.
- **Replayed responses** are stored as `TEXT` (not `jsonb`) so the replay is byte-identical.
- **Idempotency retention** — expired keys are deleted lazily when reused, or in bulk via `make idempotency-prune` (run it from cron/CronJob). The `created_at` index keeps pruning cheap.
- **In-flight duplicates** — `DB_LOCK_TIMEOUT` bounds how long a transaction waits for a row lock; a duplicate that waits too long is answered with `409 REQUEST_IN_PROGRESS` + `Retry-After` without giving up atomicity (the transaction rolls back).
