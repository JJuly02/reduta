# RedutaCTF

A self-hosted Capture The Flag platform for large events (200+ challenges,
hundreds of teams) without the operational pain of CTFd. Written in Go and
React: a single static binary plus Postgres and Redis, a denormalized
scoreboard, transactional bulk admin actions, realtime updates, and an
interface in English and Polish.

The name comes from "Reduta Ordona", Adam Mickiewicz's poem about a small
redoubt holding under an overwhelming assault: the design goal for a scoring
engine at peak load. Full documentation and screenshots are in the wiki.

**Wiki:** https://jjuly02.github.io/reduta/ (where the name comes from, how it is built, and how every part works).

## Status

All milestones (M0 to M8) complete and verified; smoke suite green at 110 checks.
The player and admin interfaces have since been reworked (progress strip,
searchable board, task modal with attempt history, live scoreboard highlight;
admin toolbar with filter/sort, select-all-matching bulk with undo, create
drawer) and the UI is localized in English and Polish.

## Quickstart

```bash
docker compose up --build         # server + worker + web + Postgres 16 + Redis 7
bash scripts/smoke.sh             # end-to-end check against the running stack
bash scripts/seed-demo.sh         # optional: a rich demo event to click around
```

- http://localhost:3000/          web UI (player + admin)
- http://localhost:8080/healthz   liveness + DB ping
- Bootstrap owner from `REDUTA_BOOTSTRAP_ADMIN_EMAIL/PASSWORD`
  (compose sets `admin@reduta.local` / `admin-dev-password` for dev).

## Layout

```
cmd/reduta-{server,worker,cli,koth-plugin}   entrypoints
internal/{config,db,store,auth}              config, pgx pool + migrations, data access, sessions
internal/core/{flags,scoring,schedule,unlock} pure domain logic
internal/{httpserver,ws,ratelimit,observability} API, websockets, limits, logging
db/migrations                                golang-migrate SQL (0001-0007)
web/                                         React + Vite + TypeScript (Bootstrap 5, i18n)
docs/                                        the wiki (index.html) + screenshots + ADRs
scripts/{smoke.sh,seed-demo.sh,shots.cjs}    smoke test, demo seed, screenshot capture
```

## Dev without local Go

```bash
make tidy   # go mod tidy in a golang container
make test   # go test ./...
make up     # full stack
```

## License

Undecided (AGPL-3.0 vs permissive, see `docs/adr/0006-license.md`); decided
before first public release, so no `LICENSE` file is committed yet.
