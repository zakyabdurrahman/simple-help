# SimpleHelp

A helpdesk application to handle customer requests.

## Tech Stack
- **Database:** PostgreSQL 16 (Docker)
- **Backend:** Go + Gin
- **Frontend:** Vue 3 + Vite (planned)
- **Orchestration:** Docker Compose
- **Migrations:** [golang-migrate](https://github.com/golang-migrate/migrate) (run automatically on backend startup)

## Prerequisites
- [Docker](https://www.docker.com/) (with Compose v2)

## First-time setup

Copy the env template and set a JWT secret (required — the backend won't
start without it):
```bash
cp .env.example .env
# edit .env and set JWT_SECRET to a strong random string
```
`.env` is gitignored; `.env.example` is the tracked template. Docker Compose
reads `.env` automatically.

## Running the App

```bash
# start db + backend (detached)
docker compose up -d

# check both are healthy
docker compose ps
```

The backend container runs pending migrations (`migrate up`) on startup
before serving traffic, then listens on `:8080`.

Verify the schema was created:
```bash
docker compose exec db psql -U simplehelp -d simplehelp -c "\dt"
```
You should see five tables: `users`, `status`, `tickets`, `comments`, and
`schema_migrations` (golang-migrate's version-tracking table).

### Connection details
| Setting | Value |
| --- | --- |
| Host (from host machine) | `localhost` |
| Port (from host machine) | `5433` |
| Port (from other containers) | `5432` |
| Database | `simplehelp` |
| User | `simplehelp` |
| Password | `simplehelp` |

Connect from a host tool (psql, DBeaver, etc.):
```bash
psql -h localhost -p 5433 -U simplehelp -d simplehelp
```

### Seeding users

A seeder command (`/app/seed`) creates one user per role (`requester`,
`agent`, `admin`), each with the shared password `password` (bcrypt-hashed,
cost 12). It is idempotent — re-running skips users that already exist.

```bash
docker compose exec backend /app/seed
```

Seed accounts:

| Email                    | Role      | Password   |
| ------------------------ | --------- | ---------- |
| `requester@example.com`  | requester | `password` |
| `agent@example.com`      | agent     | `password` |
| `admin@example.com`      | admin     | `password` |

### Configuration

The backend reads configuration from environment variables (defaults match
docker-compose):

| Variable     | Default              | Notes                                   |
| ------------ | -------------------- | --------------------------------------- |
| `DB_HOST`    | `db`                 |                                         |
| `DB_PORT`    | `5432`               |                                         |
| `DB_USER`    | `simplehelp`         |                                         |
| `DB_PASS`    | `simplehelp`         |                                         |
| `DB_NAME`    | `simplehelp`         |                                         |
| `PORT`       | `8080`               | HTTP listen port.                       |
| `JWT_SECRET` | `dev-secret-change-me` | Set to a strong random value in prod. |
| `JWT_EXPIRY` | `24h`                | Go duration syntax (e.g. `1h`, `30m`).  |

### Stopping
```bash
docker compose down          # keep data
docker compose down -v       # wipe data volume
```

## Running Tests

Integration tests live in `backend/integration/` and run against the real
PostgreSQL `db` service via a dedicated `test` compose service (under the
`test` profile). The test stage applies migrations, then runs `go test ./...`.

```bash
# ensure db is up, then build & run the test container
docker compose --profile test run --rm test
```

To pass extra flags to `go test`, override the command — but run migrations
first so the schema is in place:
```bash
docker compose --profile test run --rm test sh -c "\
  migrate -path /src/db/migrations \
    -database 'postgres://\$DB_USER:\$DB_PASS@\$DB_HOST:\$DB_PORT/\$DB_NAME?sslmode=disable' up \
  && go test -run TestRegister -v ./..."
```

> Note: the default `test` service command already runs `migrate up` then
> `go test -v ./...`. Only override the command if you need to (e.g. filter
> tests) — and remember to apply migrations in your override.

## Migrations

Migrations are SQL files in `backend/db/migrations/` using the
golang-migrate naming convention (`NNNNNN_name.up.sql` / `.down.sql`).
They are baked into the backend image and run by `entrypoint.sh` via the
`migrate` CLI on every backend startup (idempotent — only pending
migrations apply).

To add a migration, create a new numbered pair under
`backend/db/migrations/` and rebuild the backend image:
```bash
docker compose build backend
docker compose up -d backend
```

## API

See [`api.md`](api.md) for the full HTTP API contract.

## Project Structure
```
.
├── docker-compose.yml
├── spec.md
├── .env.example                   # env template (tracked)
├── .env                           # local secrets (gitignored)
├── .gitignore
├── api.md                          # API contract
├── backend/
│   ├── Dockerfile
│   ├── entrypoint.sh               # runs migrate up, then ./server
│   ├── go.mod
│   ├── cmd/
│   │   ├── server/main.go          # API server entrypoint
│   │   └── seed/main.go            # seeder command
│   ├── db/
│   │   └── migrations/
│   │       ├── 000001_init.up.sql
│   │       ├── 000001_init.down.sql
│   │       ├── 000002_status_seed.up.sql
│   │       └── 000002_status_seed.down.sql
│   ├── integration/                # integration tests
│   └── src/
│       ├── auth/                   # JWT helpers
│       ├── config/
│       ├── controllers/
│       ├── db/
│       ├── middleware/             # auth + role middleware
│       ├── models/
│       ├── repositories/
│       ├── server/
│       ├── services/
│       └── validation/
└── frontend/                       # planned
```
## Assumptions

Backend implementation assumptions (complements `spec.md`):

### Auth
- **JWT bearer tokens**, HS256-signed, default 24h expiry. Single shared secret (`JWT_SECRET`); no key rotation or refresh tokens.
- `JWT_SECRET` is **required** in Docker Compose (fails fast via `.env`); `config.go` retains a dev default only for running the binary directly outside compose (`go run`).
- No HTTPS/TLS termination in the app — assumes a reverse proxy handles TLS in production.
- No rate limiting on auth endpoints (`/login`, `/register`).
- Password hashing: bcrypt cost 12.

### Users & roles
- Roles are stored as `TEXT` with a `CHECK` constraint (`requester`, `agent`, `admin`) — **not** a separate `roles` table.
- Public registration (`POST /api/auth/register`) **always** creates a `requester`. Agent/admin accounts are created via the seeder or by an admin via `PATCH /api/users/{id}/role`.
- Email is lowercased + trimmed before the uniqueness check so case variants collide correctly.
- Email uniqueness is enforced at the application layer (check-then-insert) with the DB `UNIQUE` constraint as a backstop for race conditions; no explicit transaction wraps the check-then-insert.

### Tickets
- Requesters see **only their own** tickets; agents and admins see all. Enforced in both `GET /api/tickets` (list) and `GET /api/tickets/{id}` (detail).
- New tickets start with status `open`, default priority `medium`.
- Status transitions are enforced per the table in `api.md`/`spec.md`; illegal transitions return `422 invalid_transition`.
- Priority and assignee changes are **agent/admin only**; requesters get `403`.
- `assignee_id` is nullable; `0` in a PATCH means unassign.
- `category` is nullable free text.
- Pagination defaults: tickets `limit=20` (max 100), users `limit=50` (max 200).

### Comments
- Returned inline with ticket detail (`GET /api/tickets/{id}`) — no separate comment-list endpoint.
- Internal notes (`is_internal: true`) are **agent/admin only**; requesters get `403`.

### Error envelope
- All errors use `{"error": "<code>", "fields": {...}}`; `fields` is omitted when absent.
- Email-already-registered returns `409` with `error: "email_exists"` (not `validation_failed` as the `spec.md` example illustrated — the code's behavior is canonical).
- Invalid status transition returns `422` (not `400`).

### Migrations & seeder
- Migrations run **on every backend startup** via `migrate up` (idempotent — only pending migrations apply). The DB owns version tracking via `schema_migrations`.
- The seeder (`/app/seed`) is a separate binary run on demand; idempotent (skips existing users). The seeded password `"password"` (no digit) deliberately bypasses register validation by inserting via the repository directly.

### Tests
- Integration tests run against the **real** `db` service (no testcontainers, no mocking). Each test cleans `comments → tickets → users` to satisfy FKs and stay isolated.
- Test JWT secret/expiry are fixed constants in the test helper for deterministic, parseable tokens.

### Deployment
- Single-instance assumption — no horizontal-scaling concerns addressed (DB pool is shared; no distributed locking).
- DB credentials are hardcoded in `docker-compose.yml` (not moved to `.env`) by explicit decision; only `JWT_SECRET` is externalized.
