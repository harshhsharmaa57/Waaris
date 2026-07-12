ALTER TABLE waaris.verification_responses
    ALTER COLUMN actor_user_id DROP NOT NULL;

ALTER TABLE waaris.verification_responses
    DROP CONSTRAINT verification_responses_actor_user_id_fkey,
    ADD CONSTRAINT verification_responses_actor_user_id_fkey
        FOREIGN KEY (actor_user_id) REFERENCES waaris.users(id) ON DELETE SET NULL;

CREATE INDEX verification_requests_pending_created_idx
    ON waaris.verification_requests (created_at ASC)
    WHERE status = 'pending';

CREATE INDEX verification_responses_request_trustee_responded_idx
    ON waaris.verification_responses (request_id, trustee_id, responded_at DESC);

CREATE INDEX trustees_email_will_idx
    ON waaris.trustees (lower(email), will_id);

CREATE INDEX digital_wills_dormancy_scan_idx
    ON waaris.digital_wills (status, lifecycle_state, COALESCE(last_heartbeat_at, updated_at))
    WHERE deleted_at IS NULL;
