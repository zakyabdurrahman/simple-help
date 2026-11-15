-- 000002_status_seed.down.sql
-- Removes seeded statuses. Safe to run; rows referenced by tickets will
-- fail the FK and should be removed manually before down migration.

DELETE FROM status WHERE text IN ('open', 'in_progress', 'resolved', 'closed');
