-- 000001_init.down.sql
-- Reverses 000001_init.up.sql.

DROP TRIGGER IF EXISTS trg_tickets_updated_at ON tickets;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS status;
DROP TABLE IF EXISTS users;
