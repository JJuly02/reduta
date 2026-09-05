-- Reusable challenge library + immutable revisions (spec 4.2).
CREATE TABLE challenges (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug        TEXT NOT NULL,
    title       TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'misc',
    difficulty  SMALLINT,
    author      TEXT,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    current_rev INT  NOT NULL DEFAULT 1,
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, slug)
);
CREATE INDEX challenges_tags_idx ON challenges USING GIN (tags);

CREATE TABLE challenge_revisions (
    challenge_id   UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    rev            INT  NOT NULL,
    description_md TEXT NOT NULL DEFAULT '',
    scoring        JSONB NOT NULL DEFAULT '{"type":"static","points":100}',
    flags          JSONB NOT NULL DEFAULT '[]',   -- [{"hash":"<hex sha256>","case_sensitive":bool}]
    connection_tpl TEXT,
    created_by     UUID REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (challenge_id, rev)
);

CREATE TABLE blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    position    INT  NOT NULL DEFAULT 0,
    color       TEXT,
    schedule    JSONB,
    unlock_rule JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, name)
);

-- Embed library revisions + blocks into events (spec DDL).
ALTER TABLE event_challenges
    ADD COLUMN block_id     UUID REFERENCES blocks(id) ON DELETE SET NULL,
    ADD COLUMN challenge_id UUID REFERENCES challenges(id),
    ADD COLUMN rev          INT,
    ADD COLUMN tags         TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN schedule     JSONB,
    ADD COLUMN unlock_rule  JSONB;
CREATE INDEX event_challenges_block_idx ON event_challenges (event_id, state, block_id);
CREATE INDEX event_challenges_tags_idx  ON event_challenges USING GIN (tags);

CREATE TABLE saved_views (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    filter     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id, name)
);

-- Append-only audit trail for admin actions (spec 5.9).
CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    org_id     UUID,
    event_id   UUID,
    actor_id   UUID,
    action     TEXT NOT NULL,
    target     TEXT,
    meta       JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_event_idx ON audit_log (event_id, created_at DESC);

-- Bulk action records + undo snapshots (spec 5.5).
CREATE TABLE bulk_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    actor_id     UUID,
    action       TEXT NOT NULL,
    affected     INT  NOT NULL DEFAULT 0,
    undo_payload JSONB,
    undone       BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
