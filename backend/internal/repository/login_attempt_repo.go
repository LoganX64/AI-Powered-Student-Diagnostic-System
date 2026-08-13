package repository

import (
	"database/sql"
	"log"
	"time"
)

const maxLoginFailures = 5
const loginLockDuration = 15 * time.Minute

type LoginAttemptRepo struct {
	db *sql.DB
}

func NewLoginAttemptRepo(db *sql.DB) *LoginAttemptRepo {
	return &LoginAttemptRepo{db: db}
}

// IsLocked reports whether the account identifier is currently locked out.
func (r *LoginAttemptRepo) IsLocked(identifier string) (bool, error) {
	var lockedUntil sql.NullTime
	err := r.db.QueryRow(
		`SELECT locked_until FROM login_attempts WHERE account_identifier = $1`,
		identifier,
	).Scan(&lockedUntil)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		log.Printf("[LOGIN_ATTEMPT] IsLocked query failed for %s: %v", identifier, err)
		return false, err
	}
	return lockedUntil.Valid && lockedUntil.Time.After(time.Now()), nil
}

// RecordFailure increments the failure counter and locks the account after
// maxLoginFailures attempts within the lock window.
func (r *LoginAttemptRepo) RecordFailure(identifier string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO login_attempts (account_identifier, failures, last_failure, locked_until)
		VALUES ($1, 1, $2, NULL)
		ON CONFLICT (account_identifier) DO UPDATE
		SET failures = login_attempts.failures + 1,
		    last_failure = $2,
		    locked_until = CASE
		        WHEN login_attempts.failures + 1 >= $3 THEN $2 + $4
		        ELSE login_attempts.locked_until
		    END
	`, identifier, now, maxLoginFailures, loginLockDuration)
	if err != nil {
		log.Printf("[LOGIN_ATTEMPT] RecordFailure failed for %s: %v", identifier, err)
		return err
	}
	return nil
}

// Reset clears the failure counter and lock for a successfully authenticated account.
func (r *LoginAttemptRepo) Reset(identifier string) error {
	_, err := r.db.Exec(
		`DELETE FROM login_attempts WHERE account_identifier = $1`,
		identifier,
	)
	if err != nil {
		log.Printf("[LOGIN_ATTEMPT] Reset failed for %s: %v", identifier, err)
		return err
	}
	return nil
}
