-- Default single-org for M1 (multi-org comes later).
INSERT INTO organizations (slug, name) VALUES ('default', 'Default Org')
ON CONFLICT (slug) DO NOTHING;

CREATE TABLE events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug              TEXT NOT NULL,
    name              TEXT NOT NULL,
    state             TEXT NOT NULL DEFAULT 'draft',   -- draft|running|ended
    starts_at         TIMESTAMPTZ,
    ends_at           TIMESTAMPTZ,
    freeze_at         TIMESTAMPTZ,
    first_blood_bonus INT  NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, slug)
);

CREATE TABLE teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    invite_code TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, name)
);
CREATE UNIQUE INDEX teams_invite_code_idx ON teams (invite_code);

CREATE TABLE memberships (
    event_id  UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id   UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member',   -- captain|member
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)             -- one team per user per event
);
CREATE INDEX memberships_team_idx ON memberships (team_id);

CREATE TABLE event_challenges (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id       UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    title          TEXT NOT NULL,
    category       TEXT NOT NULL DEFAULT 'misc',
    description_md TEXT NOT NULL DEFAULT '',
    state          TEXT NOT NULL DEFAULT 'draft',   -- draft|published|hidden
    scoring        JSONB NOT NULL DEFAULT '{"type":"static","points":100}',
    position       INT  NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX event_challenges_event_idx ON event_challenges (event_id, state);

CREATE TABLE flags (
    id             BIGSERIAL PRIMARY KEY,
    ec_id          UUID NOT NULL REFERENCES event_challenges(id) ON DELETE CASCADE,
    value_hash     BYTEA NOT NULL,                 -- sha256 of normalized flag
    case_sensitive BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX flags_ec_idx ON flags (ec_id);

CREATE TABLE submissions (
    id           BIGSERIAL PRIMARY KEY,
    event_id     UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    ec_id        UUID NOT NULL REFERENCES event_challenges(id) ON DELETE CASCADE,
    team_id      UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value_hash   BYTEA NOT NULL,                   -- never store raw wrong flags
    correct      BOOLEAN NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip           INET
);
CREATE INDEX submissions_team_idx ON submissions (event_id, team_id, submitted_at DESC);
CREATE UNIQUE INDEX solves_unique ON submissions (ec_id, team_id) WHERE correct;

CREATE TABLE score_events (
    id          BIGSERIAL PRIMARY KEY,
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,                     -- solve|hint|award|penalty|adjustment
    source      TEXT NOT NULL DEFAULT 'core',
    ref_id      TEXT,
    points      INT  NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    meta        JSONB NOT NULL DEFAULT '{}',
    UNIQUE (event_id, source, ref_id)              -- idempotent writes (also from plugins)
);
CREATE INDEX score_events_event_idx ON score_events (event_id, occurred_at DESC);

CREATE TABLE scoreboard_entries (
    event_id      UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id       UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    points        INT  NOT NULL DEFAULT 0,
    solves        INT  NOT NULL DEFAULT 0,
    last_solve_at TIMESTAMPTZ,
    PRIMARY KEY (event_id, team_id)
);
CREATE INDEX scoreboard_rank_idx ON scoreboard_entries (event_id, points DESC, last_solve_at ASC);

CREATE TABLE sessions (
    token_hash BYTEA PRIMARY KEY,                  -- sha256 of the cookie token
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
