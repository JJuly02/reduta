-- Teams become global (org-scoped), CTFd-style. Admin assigns teams to events.
ALTER TABLE teams ADD COLUMN org_id UUID;
UPDATE teams SET org_id = (SELECT id FROM organizations WHERE slug='default') WHERE org_id IS NULL;
ALTER TABLE teams ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE teams ADD CONSTRAINT teams_org_fk FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE teams ALTER COLUMN event_id DROP NOT NULL;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS teams_event_id_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS teams_org_name_uniq ON teams (org_id, name);

-- One global team per user.
CREATE TABLE team_members (
    user_id   UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    team_id   UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX team_members_team_idx ON team_members (team_id);

-- Admin-managed event participation.
CREATE TABLE event_teams (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id  UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, team_id)
);
CREATE INDEX event_teams_team_idx ON event_teams (team_id);

DROP TABLE IF EXISTS memberships;
