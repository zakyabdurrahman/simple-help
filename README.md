# SimpleHelp

A helpdesk application to handle customer requests.

## Tech Stack
- **Database:** PostgreSQL 16 (Docker)
- **Backend:** Go + Gin (planned)
- **Frontend:** Vue 3 + Vite (planned)
- **Orchestration:** Docker Compose

## Prerequisites
- [Docker](https://www.docker.com/) (with Compose v2)

## Running the Database

The database is the only service containerized so far. The schema is
auto-initialized on first start via Postgres' `/docker-entrypoint-initdb.d`
mechanism (bind-mounted from `backend/internal/db/migrations`).

```bash
# start the db container (detached)
docker compose up -d db

# check it's healthy
docker compose ps

# verify the schema was created
docker compose exec db psql -U simplehelp -d simplehelp -c "\dt"
```

You should see four tables: `users`, `status`, `tickets`, `comments`.

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

### Re-running the schema migration

The initdb scripts only run on **first initialization** (when the data volume
is empty). If you edit `backend/internal/db/migrations/001_init.sql` and want
it re-applied:

```bash
# WARNING: destroys all data in the volume
docker compose down -v
docker compose up -d db
```

### Stopping
```bash
docker compose down          # keep data
docker compose down -v       # wipe data volume
```

## Project Structure
```
.
├── docker-compose.yml
├── spec.md
├── backend/
│   └── internal/
│       └── db/
│           └── migrations/
│               └── 001_init.sql   # schema (auto-run on first db init)
└── frontend/                       # planned
```
