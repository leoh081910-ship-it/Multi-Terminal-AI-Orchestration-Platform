-- seed_reviewer.sql (idempotent)
-- Register the Reviewer HTTP Runner in Go's SQLite database.
-- Safe to run multiple times — INSERT OR REPLACE prevents duplicates.
--
-- Usage:
--   sqlite3 ai-orchestration.db < scripts/seed_reviewer.sql
--   python seed_reviewer.py  (recommended — includes verification)
--
-- After running: restart Go server to load into runnerRegistry,
-- or use POST /api/v1/orgs/{orgID}/agents for hot-reload.

INSERT OR REPLACE INTO agents (
    id, org_id, name, type, status, specialties, config, runner_type, runner_config, created_at
) VALUES (
    'Reviewer',
    'default-org',
    'Reviewer',
    'Reviewer',
    'idle',
    '[]',
    '{}',
    'http',
    '{"endpoint": "http://127.0.0.1:8765/api/runner/review", "method": "POST", "timeout_ms": 600000, "output_path": "output"}',
    datetime('now')
);
