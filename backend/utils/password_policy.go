package utils

import "errors"

const MinPasswordLength = 8

var ErrPasswordTooShort = errors.New("Password must be at least 8 characters")

// ValidatePassword enforces the minimum password length. The message matches
// the frontend zod rule (validations.ts) so clients get consistent errors.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}
