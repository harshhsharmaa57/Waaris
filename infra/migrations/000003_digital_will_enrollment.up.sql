CREATE TABLE waaris.digital_wills (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('draft', 'published')),
    current_version INTEGER NOT NULL CHECK (current_version >= 1),
    dormancy_days INTEGER NOT NULL CHECK (dormancy_days BETWEEN 1 AND 3650),
    grace_days INTEGER NOT NULL CHECK (grace_days BETWEEN 1 AND 365),
    policy_version TEXT NOT NULL,
    consent_accepted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX digital_wills_active_user_uidx
    ON waaris.digital_wills (user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX digital_wills_status_updated_idx
    ON waaris.digital_wills (status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE waaris.will_release_preferences (
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('financial', 'private', 'community_shareable')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (will_id, category)
);

CREATE TABLE waaris.will_versions (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version >= 1),
    status TEXT NOT NULL CHECK (status IN ('draft', 'published')),
    dormancy_days INTEGER NOT NULL CHECK (dormancy_days BETWEEN 1 AND 3650),
    grace_days INTEGER NOT NULL CHECK (grace_days BETWEEN 1 AND 365),
    policy_version TEXT NOT NULL,
    consent_accepted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (will_id, version)
);

CREATE INDEX will_versions_will_version_idx
    ON waaris.will_versions (will_id, version DESC);

CREATE TABLE waaris.will_version_release_preferences (
    will_version_id UUID NOT NULL REFERENCES waaris.will_versions(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('financial', 'private', 'community_shareable')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (will_version_id, category)
);

CREATE TABLE waaris.consent_records (
    id UUID PRIMARY KEY,
    will_id UUID NOT NULL REFERENCES waaris.digital_wills(id) ON DELETE CASCADE,
    will_version_id UUID NOT NULL REFERENCES waaris.will_versions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    policy_version TEXT NOT NULL,
    consent_type TEXT NOT NULL CHECK (consent_type = 'digital_will_terms'),
    accepted_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX consent_records_will_accepted_idx
    ON waaris.consent_records (will_id, accepted_at DESC);
