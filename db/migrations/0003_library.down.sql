DROP TABLE IF EXISTS bulk_jobs;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS saved_views;
ALTER TABLE event_challenges
    DROP COLUMN IF EXISTS block_id,
    DROP COLUMN IF EXISTS challenge_id,
    DROP COLUMN IF EXISTS rev,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS schedule,
    DROP COLUMN IF EXISTS unlock_rule;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS challenge_revisions;
DROP TABLE IF EXISTS challenges;
