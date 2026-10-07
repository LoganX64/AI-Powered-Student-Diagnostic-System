package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// fixtures_test.go centralizes the tenant-scoped fixture graph that the
// repository tests need: tenant → admin user, coach user(+coach row), subject,
// test, questions, student, assignment. Every helper deletes only what it
// created, scoped to its own throwaway tenant, so dev rows are never touched.

// createTenant inserts a throwaway tenant and returns its id. The caller's
// t.Cleanup cascade-deletes the whole graph beneath it.
func createTenant(t *testing.T, db *sql.DB) int {
	t.Helper()
	var tid int
	if err := db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("fixtures-%s", t.Name())).Scan(&tid); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)
	})
	return tid
}

func createUser(t *testing.T, db *sql.DB, tenantID int, email, role string) int {
	t.Helper()
	var uid int
	err := db.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, 'x', $3) RETURNING id`,
		tenantID, email, role).Scan(&uid)
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return uid
}

func uniqueEmail(t *testing.T, prefix string) string {
	name := strings.NewReplacer(" ", "_", "/", "_").Replace(t.Name())
	return fmt.Sprintf("%s-%s@fixtures.local", prefix, name)
}

func createCoach(t *testing.T, db *sql.DB, tenantID int) (coachUserID, coachID int) {
	t.Helper()
	coachUserID = createUser(t, db, tenantID, uniqueEmail(t, "coach"), "coach")
	if err := db.QueryRow(`INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, 'Coach One') RETURNING id`,
		tenantID, coachUserID).Scan(&coachID); err != nil {
		t.Fatalf("create coach: %v", err)
	}
	return coachUserID, coachID
}

func createSubject(t *testing.T, db *sql.DB, tenantID int, name string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`INSERT INTO subjects (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, name).Scan(&id); err != nil {
		t.Fatalf("create subject: %v", err)
	}
	return id
}

func createTest(t *testing.T, db *sql.DB, tenantID, subjectID, coachID, duration int) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO tests (tenant_id, title, subject_id, coach_id, duration)
		VALUES ($1, 'Fixtures Test', $2, $3, $4) RETURNING id`,
		tenantID, subjectID, coachID, duration).Scan(&id)
	if err != nil {
		t.Fatalf("create test: %v", err)
	}
	return id
}

func createStudent(t *testing.T, db *sql.DB, tenantID, coachID int, code, name string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO students (tenant_id, coach_id, student_code, name) VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, coachID, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	return id
}

func createAssignment(t *testing.T, db *sql.DB, studentID, testID, coachID int) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO assignments (student_id, test_id, coach_id) VALUES ($1, $2, $3) RETURNING id`,
		studentID, testID, coachID).Scan(&id)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	return id
}

// questionFixture inserts a minimal valid question row for the given test.
func questionFixture(t *testing.T, db *sql.DB, testID int) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO questions (test_id, question_text, option_a, option_b, option_c, option_d,
		correct_answer, marks, neg_marks, importance, difficulty, type, concept_tag)
		VALUES ($1, 'Q?', 'a', 'b', 'c', 'd', 'A', 4, 1, 'medium', 'M', 'mcq', 'concept-x') RETURNING id`,
		testID).Scan(&id)
	if err != nil {
		t.Fatalf("create question: %v", err)
	}
	return id
}
