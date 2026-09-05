# ADR-0004 — PostgreSQL 16 + Redis 7

Postgres is the single source of truth (JSONB for challenge config). Redis is
scoreboard cache, rate limiting, WS fan-out pub/sub, and the asynq job queue.
No SQLite anywhere — tests use Postgres via testcontainers, so there is one SQL
path and zero dialect drift. See `INSTRUKCJE.md` ADR-004.
