ALTER TABLE event_challenges ADD COLUMN instance_spec JSONB;

CREATE TABLE instances (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ec_id      UUID NOT NULL REFERENCES event_challenges(id) ON DELETE CASCADE,
    team_id    UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    host       TEXT NOT NULL,
    port       INT  NOT NULL,
    status     TEXT NOT NULL DEFAULT 'running',   -- running|destroyed
    extends    INT  NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- one running instance per team per challenge
CREATE UNIQUE INDEX instances_active_uniq ON instances (ec_id, team_id) WHERE status = 'running';
