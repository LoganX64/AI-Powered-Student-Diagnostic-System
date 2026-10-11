BEGIN;

-- Bearer tokens for emailed result links.
--
-- A student receives a link in their inbox and may not be signed in when they
-- open it, so the token is the credential. That makes three properties
-- non-negotiable: it must be unguessable, it must not be recoverable from a
-- database read, and it must expire.
--
-- Only the SHA-256 hash is stored. The raw token exists solely in the email.
-- A dump of this table therefore yields no working links.
--
-- expires_at is NOT NULL and set at mint time: the lifetime is a decision, not
-- a default, so an unset expiry should never be reachable.

CREATE TABLE IF NOT EXISTS result_tokens (
    id SERIAL PRIMARY KEY,
    attempt_id INT NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (token_hash)
);

-- Releasing an already-released attempt mints a token for its attempt.
CREATE INDEX IF NOT EXISTS idx_result_tokens_attempt ON result_tokens (attempt_id);

COMMIT;