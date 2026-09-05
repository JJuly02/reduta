CREATE TABLE memberships (
    event_id  UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id   UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);
DROP TABLE IF EXISTS event_teams;
DROP TABLE IF EXISTS team_members;
DROP INDEX IF EXISTS teams_org_name_uniq;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS teams_org_fk;
ALTER TABLE teams DROP COLUMN IF EXISTS org_id;
