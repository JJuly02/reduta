-- Files attached to an event challenge, stored inline so a single JSON import can
-- carry everything (challenge + flags + files) and a single export round-trips it.
-- Bytes live in the DB as BYTEA; a per-file size cap is enforced at the API layer.
-- Deleting a challenge cascades its files.
CREATE TABLE event_challenge_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ec_id        UUID NOT NULL REFERENCES event_challenges(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size         BIGINT NOT NULL,
    sha256       BYTEA NOT NULL,
    data         BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX event_challenge_files_ec_idx ON event_challenge_files (ec_id, name);
