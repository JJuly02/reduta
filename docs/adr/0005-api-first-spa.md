# ADR-0005 — API-first, separate SPA

No server-side rendering for the admin panel or player view; a React SPA
consumes the JSON API. The public scoreboard may additionally get an
SSR/static variant for embedding. See `INSTRUKCJE.md` ADR-005.
