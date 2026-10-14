# Task Management API (Multi-User)

REST API untuk mengelola task per-user, dibangun dengan **Go** dan **clean architecture**.
Setiap user punya task miliknya sendiri, dapat meng-assign task ke anggota tim yang sama,
dan memiliki status/stage yang dapat dikustomisasi.

## Fitur

- **Authentication** — register & login, JWT (HS256) untuk endpoint terproteksi.
- **Tasks CRUD** — create, list (filter status, search judul, pagination), detail, update, soft delete.
- **Custom statuses** — tiap user punya 3 stage default (`todo`, `in_progress`, `done`) dan bisa menambah, mengubah, mengurutkan, serta menghapus step.
- **Assignment** — assign task ke diri sendiri atau anggota tim yang sama, dalam **satu transaksi** (update assignee + catat `task_logs` + kirim notifikasi; gagal salah satu → rollback).
- **Idempotency** — `POST /tasks` wajib menyertakan header `Idempotency-Key` (UUID); retry dengan key sama mengembalikan respons identik tanpa membuat task baru (window 24 jam). Replay ditandai header `X-Idempotent-Replayed: true`; duplikat yang masih diproses dibalas `409 REQUEST_IN_PROGRESS` + `Retry-After`.
- **Structured error handling** — semua error memakai format JSON `{status, code, message, timestamp}`; panic ditangkap middleware.
- **Structured logging** — satu log JSON per request berisi `request_id`, `method`, `path`, `status`, `latency_ms`; level INFO/WARN/ERROR mengikuti status.
- **Swagger UI** — dokumentasi interaktif di `/docs`.

## Tech Stack

Go 1.21 · [chi](https://github.com/go-chi/chi) (router) · [pgx/v5](https://github.com/jackc/pgx) +
[sqlc](https://sqlc.dev) (query) · [golang-migrate](https://github.com/golang-migrate/migrate)
(migrasi) · [golang-jwt](https://github.com/golang-jwt/jwt) · bcrypt ·
[testify](https://github.com/stretchr/testify) · [swaggo](https://github.com/swaggo/swag) +
[swgui](https://github.com/swaggest/swgui) · [air](https://github.com/air-verse/air) (hot reload).

## Menjalankan

Prasyarat: Go 1.21+, Docker, `migrate` & `sqlc` CLI (untuk target Makefile terkait)
dan `air` (hot reload; `make install-air`).

```bash
# 1. Siapkan konfigurasi
cp .env.example .env

# 2. Nyalakan Postgres (database `taskdb` dibuat otomatis oleh container)
make docker-up          # = docker compose up -d db

# 3. Migrasi skema + data dummy
make migrate-up
make seed               # opsional: 3 user, 1 tim, 2 task contoh

# 4. Jalankan API
make dev                # hot reload via air, listen di :8080
# atau: make run
```

Buka dokumentasi API di **http://localhost:8080/docs**.

Perintah Makefile lain: `make build`, `make test`, `make test-cover`, `make lint`,
`make sqlc`, `make swag`, `make migrate-down`, `make migrate-create name=...`,
`make idempotency-prune` (bersihkan key kedaluwarsa), `make docker-down`.

### Kredensial demo (dari `make seed`)

| Email | Password |
|---|---|
| alice@example.com | password123 |
| bob@example.com   | password123 |
| carol@example.com | password123 |

Ketiganya anggota tim "Engineering" (`aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`).

## Konfigurasi (environment)

Lihat `.env.example`. `DATABASE_URL` dan `JWT_SECRET` **wajib**.

| Variabel | Default | Keterangan |
|---|---|---|
| `APP_PORT` | `8080` | Port HTTP server |
| `APP_ENV` | `development` | Label environment |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `DATABASE_URL` | — | **wajib**, DSN Postgres |
| `JWT_SECRET` | — | **wajib**, secret penandatangan JWT |
| `JWT_EXPIRY` | `24h` | Masa berlaku token |
| `DB_MAX_CONNS` | `10` | Ukuran pool maksimum |
| `DB_MIN_CONNS` | `1` | Koneksi idle minimum |
| `DB_MAX_CONN_LIFETIME` | `1h` | Umur maksimum koneksi |
| `DB_MAX_CONN_IDLE_TIME` | `30m` | Waktu idle maksimum |
| `DB_LOCK_TIMEOUT` | `2s` | Batas tunggu row lock per transaksi (duplikat in-flight → `409`) |

## Arsitektur

Layered clean architecture; dependensi **selalu mengarah ke dalam**:

```
transport (HTTP)  →  usecase  →  domain  ←  repository (postgres)
```

- `domain` — entity + interface (port) + typed error. Tidak mengimpor framework/driver apa pun.
- `usecase` — business logic dan **pemilik batas transaksi**; tidak tahu chi/pgx.
- `repository/postgres` — implementasi port domain memakai sqlc/pgx; satu-satunya tempat SQL.
- `transport/http` — handler chi, middleware, DTO. Tidak pernah menyentuh `pgx`.
- `platform` — adapter lintas-cutting (jwt, bcrypt, logger, notifier).
- `cmd/api` — composition root: wiring + graceful shutdown.

```
cmd/api/                     # entrypoint
internal/
  config/                    # load env
  domain/                    # entity, port, error
  model/                     # tipe input/output use case
  usecase/                   # business logic + transaksi
  repository/postgres/       # repo + sqlc (generated)
  transport/http/            # router, middleware, handler, dto
  platform/                  # jwt, password, logger, notifier
db/migrations/               # skema (golang-migrate)
db/queries/                  # sumber query sqlc
db/seed/                     # data dummy
docs/swagger/                # spec OpenAPI (generated)
```

### Database

7 tabel: `users`, `teams`, `team_members`, `task_statuses`, `tasks`, `task_logs`,
`idempotency_keys`. Beberapa pilihan penting:

- Login case-insensitive via `UNIQUE (lower(email))`.
- `tasks` memakai **soft delete** (`deleted_at`); semua read memakai partial index `WHERE deleted_at IS NULL`.
- Search judul memakai index **GIN `pg_trgm`**.
- `UNIQUE (idempotency_keys.key)` adalah penjamin anti-duplikat saat request bersamaan.
- `tasks.status_id` → `task_statuses` dengan `ON DELETE RESTRICT` (status terpakai tidak bisa dihapus).

## API

Base path: **`/v1`** (kanonik). Health probe tidak di-version.

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| GET | `/healthz` | – | Liveness |
| GET | `/readyz` | – | Readiness (ping DB) |
| GET | `/docs` | – | Swagger UI |
| POST | `/v1/auth/register` | – | Daftar user |
| POST | `/v1/auth/login` | – | Login → JWT |
| GET | `/v1/statuses` | Bearer | List status milik user |
| POST | `/v1/statuses` | Bearer | Tambah step |
| PUT | `/v1/statuses/{id}` | Bearer | Ubah / ubah urutan step |
| DELETE | `/v1/statuses/{id}` | Bearer | Hapus step |
| POST | `/v1/tasks` | Bearer | Buat task (**wajib** header `Idempotency-Key`) |
| GET | `/v1/tasks` | Bearer | List (`?status=&search=&page=&limit=`) |
| GET | `/v1/tasks/{id}` | Bearer | Detail |
| PUT | `/v1/tasks/{id}` | Bearer | Update |
| DELETE | `/v1/tasks/{id}` | Bearer | Soft delete |
| POST | `/v1/tasks/{id}/assign` | Bearer | Assign ke anggota tim |

Endpoint terproteksi memerlukan header `Authorization: Bearer <token>`.

Format error seragam:

```json
{ "status": 400, "code": "VALIDATION_ERROR", "message": "title is required", "timestamp": "2026-01-01T00:00:00Z" }
```

### Contoh

```bash
# Register + login
curl -X POST localhost:8080/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"dave@example.com","password":"password123","name":"Dave"}'

TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}' | jq -r .access_token)

TEAM=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa

# Create task (idempotent)
curl -X POST localhost:8080/v1/tasks -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" -H 'Content-Type: application/json' \
  -d "{\"team_id\":\"$TEAM\",\"title\":\"Ship auth\"}"

# List + filter + search + pagination
curl "localhost:8080/v1/tasks?status=todo&search=ship&page=1&limit=10" -H "Authorization: Bearer $TOKEN"

# Assign ke anggota tim
curl -X POST localhost:8080/v1/tasks/<TASK_ID>/assign -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"assignee_id":"22222222-2222-2222-2222-222222222222"}'
```

## Testing

```bash
make test          # go test ./... -race -count=1
make test-cover    # + laporan coverage
```

Unit test berjalan **tanpa database/service eksternal** (fake in-memory). Fokus utama:

- **Idempotency**: request sekuensial dengan key sama → satu task & respons identik; key sama beda payload → `409`.
- **Concurrency**: N goroutine dengan key sama → tepat satu task terbuat (`go test -race`).

## Keputusan Desain (ringkas)

- **Idempotency di `TaskUsecase`, bukan middleware generik** — harus berbagi transaksi dengan insert task.
- **Transaksi** dipegang usecase lewat port `TxManager` (impl `sqlc.WithTx`); handler tidak melihat `pgx.Tx`.
- **Assign** menjalankan update assignee + audit log + notifikasi dalam satu transaksi (rollback penuh bila gagal).
- **Error** domain (tanpa HTTP status) dipetakan ke status HTTP di layer transport pada satu tempat.
- **Response replay** disimpan sebagai `TEXT` (bukan `jsonb`) agar byte-identik saat di-replay.
- **Retensi idempotency** — key kedaluwarsa dihapus *lazy* saat dipakai ulang, atau massal via `make idempotency-prune` (jalankan lewat cron/CronJob). Index pada `created_at` membuat prune murah.
- **In-flight duplicate** — `DB_LOCK_TIMEOUT` membatasi lama menunggu row lock; duplikat yang menunggu terlalu lama dibalas `409 REQUEST_IN_PROGRESS` + `Retry-After` tanpa mengorbankan atomisitas (transaksinya rollback).
