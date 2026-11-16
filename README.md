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

Copy the env template and fill in real values. At minimum set a strong
`JWT_SECRET` (required — the backend won't start without it). The `DB_*` and
`POSTGRES_*` variables come pre-populated with the default `simplehelp`
credentials, which are fine for local dev — change them for anything beyond
that:
```bash
cp .env.example .env
# edit .env: set JWT_SECRET (and DB creds if not using the defaults)
```
`.env` is gitignored; `.env.example` is the tracked template. Docker Compose
reads `.env` automatically for `${VAR}` substitution in `docker-compose.yml`.

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
docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "\dt"'
```
You should see five tables: `users`, `status`, `tickets`, `comments`, and
`schema_migrations` (golang-migrate's version-tracking table).

### Accessing the app
| Service   | URL                    |
| --------- | ---------------------- |
| Frontend  | http://localhost:3000  |
| Backend   | http://localhost:8080  |
| Database  | localhost:5433         |

The frontend (Nginx) proxies `/api` to the backend, so all API calls go
through a single origin — no CORS configuration needed.

### Connection details

The DB values below come from `.env` (`POSTGRES_DB`/`POSTGRES_USER`/
`POSTGRES_PASSWORD` for the `db` service, mirrored as `DB_*` for the backend).
Defaults are shown; override them in `.env` for non-default setups.

| Setting | Value (default) |
| --- | --- |
| Host (from host machine) | `localhost` |
| Port (from host machine) | `5433` |
| Port (from other containers) | `5432` |
| Database | `${POSTGRES_DB}` / `${DB_NAME}` |
| User | `${POSTGRES_USER}` / `${DB_USER}` |
| Password | `${POSTGRES_PASSWORD}` / `${DB_PASS}` |

Connect from a host tool (psql, DBeaver, etc.):
```bash
psql -h localhost -p 5433 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
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

The backend reads configuration from environment variables. Defaults below
match the in-code fallbacks so the app runs outside Docker without an env
file; inside Docker, the values come from `.env` via `docker-compose.yml`
substitution.

| Variable          | Default                | Notes                                        |
| ----------------- | ---------------------- | -------------------------------------------- |
| `DB_HOST`         | `db`                   |                                              |
| `DB_PORT`         | `5432`                 |                                              |
| `DB_USER`         | `simplehelp`           |                                              |
| `DB_PASS`         | `simplehelp`           |                                              |
| `DB_NAME`         | `simplehelp`           |                                              |
| `PORT`            | `8080`                 | HTTP listen port.                            |
| `JWT_SECRET`      | `dev-secret-change-me` | Set to a strong random value in prod.        |
| `JWT_EXPIRY`      | `24h`                  | Go duration syntax (e.g. `1h`, `30m`).       |
| `POSTGRES_DB`     | `simplehelp`           | `db` service only (initial DB created).      |
| `POSTGRES_USER`   | `simplehelp`           | `db` service only (initial superuser).       |
| `POSTGRES_PASSWORD` | `simplehelp`          | `db` service only. Change in `.env` for dev. |

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
├── postman_collection.json         # Postman collection
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
└── frontend/
    ├── Dockerfile                  # multi-stage: Node build → Nginx serve
    ├── nginx.conf                  # proxies /api → backend
    ├── package.json
    ├── vite.config.ts
    ├── tailwind.config.js
    └── src/
        ├── api/                    # axios client + interceptors
        ├── assets/                 # Tailwind CSS
        ├── components/             # StatusBadge, FormField
        ├── router/                 # Vue Router with auth guards
        ├── stores/                 # Pinia auth store
        ├── types/                  # TS types matching API models
        └── views/                  # Login, TicketList, TicketCreate, TicketDetail, Users
```
## Assumptions

