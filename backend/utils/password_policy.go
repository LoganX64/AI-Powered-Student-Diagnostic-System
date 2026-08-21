package utils

import "errors"

const minPasswordLength = 8

var errPasswordTooShort = errors.New("Password must be at least 8 characters")

// ValidatePassword enforces the minimum password length. The message matches
// the frontend zod rule (validations.ts) so clients get consistent errors.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return errPasswordTooShort
	}
	return nil
}
