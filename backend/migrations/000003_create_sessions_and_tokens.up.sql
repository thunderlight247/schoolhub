-- Section 27: session strategy, password reset architecture, account
-- invitation architecture. Email *sending* is out of scope here — these
-- tables and the tokens that reference them are the architecture; the
-- actual email delivery is stubbed (logged, not sent) until a real
-- provider is wired up.

-- ---------------------------------------------------------------------------
-- sessions: server-side session strategy (chosen over JWTs specifically so
-- that logout and suspending a user take effect immediately — revoking a
-- session is just deleting a row, no blocklist required).
--
-- Only a SHA-256 hash of the session token is ever stored. The raw token
-- lives in an HttpOnly cookie on the client and nowhere else; a database
-- leak alone does not yield usable session tokens.
-- ---------------------------------------------------------------------------
CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    user_agent  TEXT,
    ip_address  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

-- ---------------------------------------------------------------------------
-- password_reset_tokens: single-use, expiring, hashed the same way as
-- session tokens.
-- ---------------------------------------------------------------------------
CREATE TABLE password_reset_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);

-- ---------------------------------------------------------------------------
-- invitation_tokens: lets an admin create a `people` row for a new staff
-- member (or later, a guardian) and invite them to set their own password
-- and activate their `users` account, rather than the admin choosing a
-- password on their behalf. role_id is fixed at invite time by whoever
-- sends it (section 12: "Assign system role").
-- ---------------------------------------------------------------------------
CREATE TABLE invitation_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    role_id     UUID NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    invited_by  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ
);

CREATE INDEX idx_invitation_tokens_person_id ON invitation_tokens (person_id);
