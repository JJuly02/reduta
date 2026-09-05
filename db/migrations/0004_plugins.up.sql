CREATE TABLE plugins (
    id           TEXT PRIMARY KEY,               -- e.g. 'koth'
    name         TEXT NOT NULL,
    base_url     TEXT,
    secret       TEXT NOT NULL,                  -- HMAC secret for outbound webhooks
    token_hash   BYTEA NOT NULL,                 -- sha256 of the plg_ bearer token
    capabilities JSONB NOT NULL DEFAULT '[]',
    events       JSONB NOT NULL DEFAULT '[]',    -- subscribed core event types
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
