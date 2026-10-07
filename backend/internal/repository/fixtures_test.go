package repository

import (
	"database/sql"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// The tenant-graph fixtures live in internal/testutil so the handler tests can
// build the same graph (see internal/handler/handler_fixtures_test.go). These
// are thin aliases kept so the repository tests read unchanged; the repo-only
// fixture `attemptFixture` lives in attempt_repo_exam_test.go.

func createTenant(t *testing.T, db *sql.DB) int { return testutil.CreateTenant(t, db) }

func createUser(t *testing.T, db *sql.DB, tenantID int, email, role string) int {
	return testutil.CreateUser(t, db, tenantID, email, role)
}

func uniqueEmail(t *testing.T, prefix string) string { return testutil.UniqueEmail(t, prefix) }

func createCoach(t *testing.T, db *sql.DB, tenantID int) (coachUserID, coachID int) {
	return testutil.CreateCoach(t, db, tenantID)
}

func createSubject(t *testing.T, db *sql.DB, tenantID int, name string) int {
	return testutil.CreateSubject(t, db, tenantID, name)
}

func createTest(t *testing.T, db *sql.DB, tenantID, subjectID, coachID, duration int) int {
	return testutil.CreateTest(t, db, tenantID, subjectID, coachID, duration)
}

func createStudent(t *testing.T, db *sql.DB, tenantID, coachID int, code, name string) int {
	return testutil.CreateStudent(t, db, tenantID, coachID, code, name)
}

func createAssignment(t *testing.T, db *sql.DB, studentID, testID, coachID int) int {
	return testutil.CreateAssignment(t, db, studentID, testID, coachID)
}

func questionFixture(t *testing.T, db *sql.DB, testID int) int {
	return testutil.CreateQuestion(t, db, testID)
}
