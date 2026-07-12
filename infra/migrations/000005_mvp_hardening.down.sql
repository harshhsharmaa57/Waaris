DROP INDEX IF EXISTS waaris.digital_wills_dormancy_scan_idx;
DROP INDEX IF EXISTS waaris.trustees_email_will_idx;
DROP INDEX IF EXISTS waaris.verification_responses_request_trustee_responded_idx;
DROP INDEX IF EXISTS waaris.verification_requests_pending_created_idx;

ALTER TABLE waaris.verification_responses
    DROP CONSTRAINT verification_responses_actor_user_id_fkey,
    ALTER COLUMN actor_user_id SET NOT NULL,
    ADD CONSTRAINT verification_responses_actor_user_id_fkey
        FOREIGN KEY (actor_user_id) REFERENCES waaris.users(id) ON DELETE CASCADE;
