package repository

import (
	"database/sql"
	"time"

	"ai-student-diagnostic/backend/utils"
)

// ResultTokenRow is a stored bearer token for one released attempt.
//
// Only the hash is persisted. The raw token exists solely in the email the
// student receives, so a database read cannot produce a working link.
type ResultTokenRow struct {
	ID        int
	AttemptID int
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type ResultTokenRepo struct {
	DB *sql.DB
}

func NewResultTokenRepo(db *sql.DB) *ResultTokenRepo {
	return &ResultTokenRepo{DB: db}
}

// Mint generates a bearer token for an attempt and stores only its SHA-256
// hash, mirroring how password-reset tokens are handled in internal/auth.
//
// The returned raw token is the caller's responsibility to deliver and is never
// persisted. ttl is supplied rather than hardcoded so the lifetime policy is
// visible at the call site and tests can use short windows.
//
// Collisions are not handled specially: 256 bits makes a collision
// infeasible, and the UNIQUE constraint would surface one as an error anyway.
func (r *ResultTokenRepo) Mint(attemptID int, ttl time.Duration) (string, error) {
	token, err := utils.RandomTokenHex(32)
	if err != nil {
		return "", err
	}
	hash := utils.HashSHA256Hex(token)
	expiresAt := time.Now().Add(ttl)

	_, err = r.DB.Exec(
		`INSERT INTO result_tokens (attempt_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		attemptID, hash, expiresAt,
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

// Resolve maps a presented raw token to its attempt.
//
// Unknown and expired both return sql.ErrNoRows, deliberately. Distinguishing
// them would let a caller probe for whether an attempt exists, so the HTTP layer
// can render every failure as the same response.
//
// The hash is computed rather than the raw token being compared, so no raw
// token is ever used in a query.
func (r *ResultTokenRepo) Resolve(rawToken string) (int, error) {
	if rawToken == "" {
		return 0, sql.ErrNoRows
	}
	hash := utils.HashSHA256Hex(rawToken)

	var attemptID int
	err := r.DB.QueryRow(
		`SELECT attempt_id FROM result_tokens WHERE token_hash = $1 AND expires_at > NOW()`,
		hash,
	).Scan(&attemptID)
	if err != nil {
		return 0, err
	}
	return attemptID, nil
}

// DeleteForAttempt revokes every token minted for an attempt. Used when a
// result is released again so the earlier link stops working rather than two
// live links existing for the same result.
func (r *ResultTokenRepo) DeleteForAttempt(attemptID int) error {
	_, err := r.DB.Exec(`DELETE FROM result_tokens WHERE attempt_id = $1`, attemptID)
	return err
}