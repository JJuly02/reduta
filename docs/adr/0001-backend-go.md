# ADR-0001 — Backend in Go

**Status:** accepted (working title stage)

CTFd's slowness is architectural (sync gunicorn workers, N+1 queries, live
scoreboard aggregation, unvirtualized admin tables), not "because Python".
We fix the architecture and choose Go for: a single static binary (trivial
self-hosting), native Docker/K8s clients (per-team instances + KotH),
goroutines for scoring ticks and thousands of WS connections, and predictable
latency without GC stalls.

Laravel 11 + Octane/FrankenPHP is an acceptable alternative **iff** the team is
PHP-first; sections 4–7 of the spec still hold. Decision criterion is team
competence, not a benchmark. See `INSTRUKCJE.md` ADR-001.
