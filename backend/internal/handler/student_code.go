package handlers

import (
	"errors"
	"fmt"
	"math/rand"

	"ai-student-diagnostic/backend/internal/repository"

	"github.com/lib/pq"
)

const studentCodeChars = "0123456789abcdefghijklmnopqrstuvwxyz"

func generateStudentCode(tenantID int) string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = studentCodeChars[rand.Intn(len(studentCodeChars))]
	}
	return fmt.Sprintf("T%d%s", tenantID, string(b))
}

// ensureStudentCode creates a student, auto-generating a globally unique code
// (format T{tenant}{base36-6}) when none is provided, retrying on a unique
// constraint violation. Student codes are the login secret, so they must be
// unique across the whole table, not just per tenant.
func ensureStudentCode(studentRepo *repository.StudentRepo, tenantID int, name, providedCode string, coachID int) (int, string, error) {
	code := providedCode
	for attempt := 0; attempt < 10; attempt++ {
		if code == "" {
			code = generateStudentCode(tenantID)
		}
		id, err := studentRepo.Create(tenantID, name, code, coachID)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				code = "" // regenerate and retry
				continue
			}
			return 0, "", err
		}
		return id, code, nil
	}
	return 0, "", errors.New("failed to generate a unique student code")
}
