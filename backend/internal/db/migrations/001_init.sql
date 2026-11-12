-- 001_init.sql
-- Initial schema for SimpleHelp.
-- Auto-run by Postgres on first container init (empty data dir) via
-- /docker-entrypoint-initdb.d. To re-run after edits:
--   docker compose down -v && docker compose up -d db

BEGIN;

-- users ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL
                  CHECK (role IN ('requester', 'agent', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- status -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS status (
    id   SMALLSERIAL PRIMARY KEY,
    text TEXT NOT NULL UNIQUE
);

-- tickets ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tickets (
    id           BIGSERIAL PRIMARY KEY,
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL,
    category     TEXT,
    priority     TEXT        NOT NULL DEFAULT 'medium'
                 CHECK (priority IN ('low', 'medium', 'high')),
    status_id    SMALLINT    NOT NULL REFERENCES status(id),
    requester_id BIGINT      NOT NULL REFERENCES users(id),
    assignee_id  BIGINT      REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- comments -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS comments (
    id          BIGSERIAL PRIMARY KEY,
    ticket_id   BIGINT      NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id   BIGINT      NOT NULL REFERENCES users(id),
    body        TEXT        NOT NULL,
    is_internal BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- indexes on foreign keys -------------------------------------------
CREATE INDEX IF NOT EXISTS idx_tickets_status_id    ON tickets(status_id);
CREATE INDEX IF NOT EXISTS idx_tickets_requester_id ON tickets(requester_id);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee_id  ON tickets(assignee_id);
CREATE INDEX IF NOT EXISTS idx_comments_ticket_id   ON comments(ticket_id);
CREATE INDEX IF NOT EXISTS idx_comments_author_id   ON comments(author_id);

-- updated_at trigger -------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tickets_updated_at ON tickets;
CREATE TRIGGER trg_tickets_updated_at
    BEFORE UPDATE ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;
