# Changelog

All notable changes to this project are documented here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added - challenge file attachments in the import JSON
- Challenges can carry files inline in the import/export JSON: each challenge may
  include a `files` array of `{name, content_type?, data (base64)}`, so a whole set
  of challenges and their downloadable attachments loads from one JSON in a single
  import. Export emits files the same way; `?files=meta` omits the bytes.
- Migration `0008_challenge_files`: files are stored in Postgres (BYTEA) and cascade
  with their challenge. Per-file cap 25 MiB; import body cap raised to 128 MiB.
- New endpoint `GET /events/{id}/challenges/{ec}/files/{fileID}` streams a file,
  gated by the same visibility as the challenge. The challenge detail now lists its
  files and the player challenge view shows download links. CI-tolerant lint fixes
  and the `nhooyr.io/websocket -> coder/websocket` migration ship alongside.

### Changed - global teams (CTFd-style)
- Migration `0007_global_teams`: teams are now org-scoped and global (one team per
  user via `team_members`); admins assign teams to events via `event_teams`.
  Replaces per-event teams and the old `memberships` table.
- Onboarding: a player creates a global team or joins one with an invite code
  right after signing in; the captain sees the invite code to share. Players see
  only the events their team is assigned to.
- Admin: Teams section assigns/removes teams for an event; scoreboard and
  participation are gated by assignment. New endpoints: POST /teams, POST
  /teams/join, GET /me/team, admin GET /teams, POST/DELETE event-teams.

### Added - admin panel + notifications
- CTFd-style admin panel (dark section nav, light title band, white body):
  Statistics, Notifications, Events (was Pages), Challenges, Blocks, Library,
  Teams, Submissions, Scoreboard; account dropdown carries Admin Panel + Log out.
- Notifications (admin sends, players get a live toast + a notifications page).
- Scoreboard progress line chart (+ GET /scoreboard/series).

### Added - M8 (per-team instances + KotH plugin)
- Migration `0005_instances`: `instances` table + `instance_spec` on challenges.
- Instance lifecycle API (create, get, extend, destroy), one running instance per
  team per challenge, global cap, TTL. Provisioning is a mock by default (ADR-0009);
  a real Docker/K8s provisioner is the documented opt-in.
- Per-team dynamic flags (`{{team_flag}}`, HMAC per event/team/challenge) so a
  leaked flag does not validate for another team; verified on submit.
- Reference `koth-plugin` service (manifest, signed-webhook verify + dedup, awards
  the crown holder on tick) that passes `reduta-cli plugin verify` (ADR-0010).

### Added - M7 (plugin API v1)
- Migration `0004_plugins`. Admin registers plugins (`POST /plugins`), receiving a
  one-time `plg_` bearer token and webhook secret.
- Plugin API (Bearer plg_): idempotent awards `POST /plugin/v1/awards`
  (UNIQUE(event, source, ref_id), adjusts scoreboard), award reversal, team list,
  announcements (pushed over WS), config.
- Signed outbound webhooks (`X-Reduta-Signature: sha256=HMAC(secret, ts.body)`) on
  `solve.created` and a per-minute `tick.minute` emitter to subscribed plugins.
- `reduta-cli plugin verify <url>` checks a plugin accepts signed webhooks, is
  idempotent on redelivery, and rejects bad signatures.
- Scoreboard cache is now invalidated on every score change.

### Added - M6 (frontend)
- React 19 + Vite + TypeScript SPA (`web/`), TanStack Query, React Router 7,
  hand-written dark-first design system (ADR-0008). Served by an nginx `web`
  service that proxies `/api` and `/ws` to the server.
- Screens: sign in / register; events list (+ create for admins); player board
  grouped by blocks with open/locked/scheduled/solved states, side panel with
  description and flag submission; live scoreboard; admin (challenge table with
  multi-select bulk actions, publish/hide, create, import/export, blocks,
  library embed/clone). Player board and scoreboard update live over WebSocket.
- Player status endpoint `GET /events/{id}/me` (team, solved ids, points).


### Added — M5 (realtime + cache)
- WebSocket hub (`internal/ws`, nhooyr) at `/ws?event=`, with Redis pub/sub bridge
  for cross-instance fan-out; server publishes `scoreboard` and `challenges.changed`
  on solves and admin changes. Scoreboard responses cached in Redis (2s).

### Added — M4 (import / export)
- `GET /events/{id}/export` (JSON: event, blocks, challenges incl. hashed flags).
- `POST /events/{id}/import` native JSON with `?dry_run=true` plan
  (created/updated), idempotent by title; flags accept plaintext or hex hash.
- `POST /events/{id}/import?format=ctfd` — CTFd challenges/flags importer
  (regex flags skipped, unknown types imported as standard).

### Added — M3 (schedules + unlock rules)
- Pure evaluators `internal/core/schedule` (RRULE via rrule-go, one-off + recurring
  windows, closed_behavior hidden/locked/readonly) and `internal/core/unlock`
  (all/any/not, solved, team_points_gte, solved_count_gte, after, block_completed;
  100% tested). Unlock predicates key on event_challenge ids (documented deviation
  from spec's challenge_slug).
- Player challenge listing and submit are gated by schedule + per-team unlock, with
  block inheritance; listing exposes per-challenge `status` (open|locked|scheduled).
- Endpoints: PATCH challenge schedule/unlock, PATCH block schedule, bulk `set_schedule`.

### Added — M2 (challenge library)
- Migration `0003_library`: challenges + challenge_revisions (immutable, versioned),
  blocks, event_challenges embedding columns, saved_views, audit_log, bulk_jobs.
- Library CRUD + revisions + clone; embed-from-library (pins a revision, copies flags).
- Blocks; bulk admin actions (publish/hide/archive/assign_block/add|remove_tags/
  set_schedule/delete) applied synchronously with audit entries and undo snapshots;
  saved views.

### Added — M1 (core)
- Migration `0002_core`: events, teams, memberships, event_challenges, flags,
  submissions, score_events, scoreboard_entries, sessions (+ default org).
- Data layer `internal/store` (hand-written pgx; see ADR-0007).
- Auth: register/login/logout/me, argon2id passwords, server-side sessions
  (sha256-hashed tokens), hardened cookie; role gate (owner/admin vs player).
- Events (admin create + state), teams (create/join by invite/list), challenges
  (admin create/publish; player list/detail — never exposes flags).
- Flag submission: normalize+sha256 verify, static scoring, first-blood bonus,
  idempotent solves (partial unique index), wrong-answer hashing, in-memory
  submit rate limiting, event time-window enforcement.
- Denormalized scoreboard (`scoreboard_entries`, never aggregated per request).
- RFC 9457 `application/problem+json` errors.
- Bootstrap owner from env on first start; `scripts/smoke.sh` end-to-end test.

### Added — M0 (skeleton)
- Go modular monolith (`reduta-server` + `reduta-worker`), config, health,
  embedded golang-migrate, docker-compose (pg16+redis7), CI, Makefile.
