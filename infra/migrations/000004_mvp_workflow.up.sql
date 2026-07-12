ALTER TABLE waaris.digital_wills
    ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'active' CHECK (lifecycle_state IN ('active', 'pending_verification', 'grace_period', 'ready_for_execution')),
    ADD COLUMN last_heartbeat_at TIMESTAMPTZ,
    ADD COLUMN pending_verification_started_at TIMESTAMPTZ,
    ADD COLUMN grace_period_started_at TIMESTAMPTZ,
    ADD COLUMN ready_for_execution_at TIMESTAMPTZ;

CREATE INDEX digital_wills_lifecycle_state_idx
    ON waaris.digital_wills (lifecycle_state, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE waaris.trustees (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    relationship TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX trustees_will_email_uidx
    ON waaris.trustees (will_id, lower(email));

CREATE INDEX trustees_user_created_idx
    ON waaris.trustees (user_id, created_at DESC);

CREATE TABLE waaris.heartbeats (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX heartbeats_will_occurred_idx
    ON waaris.heartbeats (will_id, occurred_at DESC);

CREATE TABLE waaris.verification_requests (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    threshold_required INTEGER NOT NULL CHECK (threshold_required >= 1),
    status TEXT NOT NULL CHECK (status IN ('pending', 'resolved', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX verification_requests_pending_will_uidx
    ON waaris.verification_requests (will_id)
    WHERE status = 'pending';

CREATE INDEX verification_requests_user_created_idx
    ON waaris.verification_requests (user_id, created_at DESC);

CREATE TABLE waaris.verification_responses (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES waaris.verification_requests(id) ON DELETE CASCADE,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    trustee_id UUID NOT NULL REFERENCES waaris.trustees(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    decision TEXT NOT NULL CHECK (decision IN ('approve', 'reject', 'abstain')),
    responded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX verification_responses_request_responded_idx
    ON waaris.verification_responses (request_id, responded_at DESC);

CREATE TABLE waaris.notifications (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    trustee_id UUID REFERENCES waaris.trustees(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('email')),
    recipient_name TEXT NOT NULL,
    recipient_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'sent', 'failed')),
    queued_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    failure_message TEXT
);

CREATE INDEX notifications_user_queued_idx
    ON waaris.notifications (user_id, queued_at DESC);

CREATE INDEX notifications_status_queued_idx
    ON waaris.notifications (status, queued_at ASC);

CREATE TABLE waaris.audit_events (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES waaris.users(id) ON DELETE CASCADE,
    will_id UUID REFERENCES waaris.digital_wills(id) ON DELETE SET NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX audit_events_user_occurred_idx
    ON waaris.audit_events (user_id, occurred_at DESC);
