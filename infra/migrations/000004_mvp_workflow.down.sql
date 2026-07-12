DROP TABLE IF EXISTS waaris.audit_events;
DROP TABLE IF EXISTS waaris.notifications;
DROP TABLE IF EXISTS waaris.verification_responses;
DROP TABLE IF EXISTS waaris.verification_requests;
DROP TABLE IF EXISTS waaris.heartbeats;
DROP TABLE IF EXISTS waaris.trustees;
DROP INDEX IF EXISTS waaris.digital_wills_lifecycle_state_idx;
ALTER TABLE waaris.digital_wills
    DROP COLUMN IF EXISTS ready_for_execution_at,
    DROP COLUMN IF EXISTS grace_period_started_at,
    DROP COLUMN IF EXISTS pending_verification_started_at,
    DROP COLUMN IF EXISTS last_heartbeat_at,
    DROP COLUMN IF EXISTS lifecycle_state;
