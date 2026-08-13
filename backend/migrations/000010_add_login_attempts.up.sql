CREATE TABLE login_attempts (
    account_identifier TEXT PRIMARY KEY,
    failures           INT NOT NULL DEFAULT 0,
    last_failure       TIMESTAMPTZ,
    locked_until       TIMESTAMPTZ
);
