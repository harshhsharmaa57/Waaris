CREATE TABLE waaris.users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_email_lowercase CHECK (email = LOWER(email)),
    CONSTRAINT users_email_length CHECK (char_length(email) <= 254),
    CONSTRAINT users_display_name_length CHECK (char_length(display_name) <= 100)
);

CREATE TABLE waaris.refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT refresh_tokens_expiry CHECK (expires_at > created_at)
);

CREATE INDEX refresh_tokens_active_user_idx ON waaris.refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;
