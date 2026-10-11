package handlers

import (
	"errors"
	"fmt"
	"math/rand"

	"ai-student-diagnostic/backend/internal/repository"

	"github.com/lib/pq"
)

const studentCodeChars = "0123456789abcdefghijklmnopqrstuvwxyz"

// generateStudentCode is a package-level var so tests can force a collision and
// exercise ensureStudentCode's 23505 retry path. Production never reassigns it.
var generateStudentCode = func(tenantID int) string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = studentCodeChars[rand.Intn(len(studentCodeChars))]
	}
	return fmt.Sprintf("T%d%s", tenantID, string(b))
}

// ErrStudentEmailTaken is returned when a student email collides with another
// student in the same tenant. Surfaced by the handler as a 400.
var ErrStudentEmailTaken = errors.New("email already in use for another student in your organization")

// Postgres reports a UNIQUE *constraint* violation under the constraint name
// and a UNIQUE *index* violation under the index name, so these differ.
const (
	// studentCodeConstraint, from 000012 (global: codes are a login secret).
	studentCodeConstraint = "students_student_code_key"
	// studentEmailIndex, from 000021 (per-tenant, partial).
	studentEmailIndex = "idx_students_email_tenant"
)

// ensureStudentCode creates a student, auto-generating a globally unique code
// (format T{tenant}{base36-6}) when none is provided, retrying on a unique
// constraint violation. Student codes are the login secret, so they must be
// unique across the whole table, not just per tenant.
//
// The retry is scoped to the student_code constraint on purpose. Migration
// 000021 added a second unique index (idx_students_email_tenant), and before
// this check a duplicate email burned all ten attempts regenerating codes that
// could never help, then reported "failed to generate a unique student code".
func ensureStudentCode(studentRepo *repository.StudentRepo, tenantID int, name, providedCode string, coachID int, email *string) (int, string, error) {
	code := providedCode
	for attempt := 0; attempt < 10; attempt++ {
		if code == "" {
			code = generateStudentCode(tenantID)
		}
		id, err := studentRepo.Create(tenantID, name, code, coachID, email)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				switch pqErr.Constraint {
case studentEmailIndex:
					// Retrying cannot change the outcome; the email is the
					// collision, not the code.
					return 0, "", ErrStudentEmailTaken
				case studentCodeConstraint:
					code = "" // regenerate and retry
					continue
				}
			}
			return 0, "", err
		}
		return id, code, nil
	}
	return 0, "", errors.New("failed to generate a unique student code")
}
