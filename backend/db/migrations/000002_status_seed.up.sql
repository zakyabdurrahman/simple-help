-- 000002_status_seed.up.sql
-- Seed the status table with the canonical ticket statuses.
-- The text column matches the string constants used by the API.

INSERT INTO status (text) VALUES
    ('open'),
    ('in_progress'),
    ('resolved'),
    ('closed')
ON CONFLICT (text) DO NOTHING;
