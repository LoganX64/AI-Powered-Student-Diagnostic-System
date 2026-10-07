package testutil

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

// Fixtures for the tenant-scoped object graph that the repository and handler
// tests both need: tenant → admin user, coach user(+coach row), subject, test,
// questions, student, assignment.
//
// Every helper registers a t.Cleanup that deletes only the throwaway tenant it
// created. Because users/coaches/subjects/tests/students/assignments/attempts
// all cascade from tenants, that one delete removes the whole graph. Dev data is
// never touched, and tests stay independent because Go runs a package's tests
// sequentially and each cleanup fires before the next test starts.

// CreateTenant inserts a throwaway tenant and returns its id. Cleanup cascades
// to everything created beneath it.
func CreateTenant(t *testing.T, db *sql.DB) int {
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

// seq disambiguates repeated calls within one test (e.g. the primary and the
// foreign tenant both need a coach).
var seq atomic.Int64

// UniqueEmail builds a unique email address per call. t.Name() alone is not
// enough: a fixture that creates two users in the same test (primary tenant +
// foreign tenant) would otherwise collide on the unique email index.
// t.Name() can also contain spaces and slashes (subtests), which are not valid
// in an email local part.
func UniqueEmail(t *testing.T, prefix string) string {
	name := strings.NewReplacer(" ", "_", "/", "_").Replace(t.Name())
	if len(name) > 30 {
		name = name[:30]
	}
	return fmt.Sprintf("%s-%s-%d@fixtures.local", prefix, name, seq.Add(1))
}

// UniqueCode builds a short unique identifier for the columns that cap at 50
// characters (students.student_code). Uses the same sequence as UniqueEmail so
// repeated calls in one test stay distinct.
func UniqueCode(t *testing.T, prefix string) string {
	name := strings.NewReplacer(" ", "_", "/", "_").Replace(t.Name())
	if len(name) > 20 {
		name = name[:20]
	}
	return fmt.Sprintf("%s-%s-%d", prefix, name, seq.Add(1))
}

func CreateUser(t *testing.T, db *sql.DB, tenantID int, email, role string) int {
	t.Helper()
	var uid int
	err := db.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, 'x', $3) RETURNING id`,
		tenantID, email, role).Scan(&uid)
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return uid
}

func CreateAdmin(t *testing.T, db *sql.DB, tenantID int) int {
	t.Helper()
	return CreateUser(t, db, tenantID, UniqueEmail(t, "admin"), "admin")
}

// CreateCoach returns the coach's user id and coach row id.
func CreateCoach(t *testing.T, db *sql.DB, tenantID int) (coachUserID, coachID int) {
	t.Helper()
	coachUserID = CreateUser(t, db, tenantID, UniqueEmail(t, "coach"), "coach")
	if err := db.QueryRow(`INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, 'Coach One') RETURNING id`,
		tenantID, coachUserID).Scan(&coachID); err != nil {
		t.Fatalf("create coach: %v", err)
	}
	return coachUserID, coachID
}

func CreateSubject(t *testing.T, db *sql.DB, tenantID int, name string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`INSERT INTO subjects (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, name).Scan(&id); err != nil {
		t.Fatalf("create subject: %v", err)
	}
	return id
}

// CreateTest inserts a test row. subject_name is denormalised onto tests and is
// nullable in the schema, but every production write path
// (TestPaperRepo.Create) populates it, and AssignmentRepo.ListAll scans it into
// a plain string — a NULL there turns the whole assignment list into a 500. So
// the fixture must set it too, or handler tests that list assignments fail for
// a reason that does not exist in production.
func CreateTest(t *testing.T, db *sql.DB, tenantID, subjectID, coachID, duration int) int {
	t.Helper()
	return CreateTestNamed(t, db, tenantID, subjectID, coachID, duration, "Fixtures Test", "Fixtures Subject")
}

// CreateTestNamed is CreateTest with explicit title and denormalised
// subject_name.
func CreateTestNamed(t *testing.T, db *sql.DB, tenantID, subjectID, coachID, duration int, title, subjectName string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO tests (tenant_id, title, subject_id, coach_id, duration, subject_name)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		tenantID, title, subjectID, coachID, duration, subjectName).Scan(&id)
	if err != nil {
		t.Fatalf("create test: %v", err)
	}
	return id
}

func CreateStudent(t *testing.T, db *sql.DB, tenantID, coachID int, code, name string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO students (tenant_id, coach_id, student_code, name) VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, coachID, code, name).Scan(&id)
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	return id
}

func CreateAssignment(t *testing.T, db *sql.DB, studentID, testID, coachID int) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO assignments (student_id, test_id, coach_id) VALUES ($1, $2, $3) RETURNING id`,
		studentID, testID, coachID).Scan(&id)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	return id
}

// CreateQuestion inserts a minimal valid question row for the given test.
//
// importance/difficulty/type use the values the current schema constrains
// (000007 widened importance to high/medium/low and type to mcq/multi/integer).
// expected_time and concept_tag are nullable in the schema but are scanned into
// plain non-nullable types by TestPaperRepo.ListQuestions, which the SQI
// computation path uses — so the fixture must populate them or any test that
// computes an attempt's SQI fails with a NULL conversion error.
func CreateQuestion(t *testing.T, db *sql.DB, testID int) int {
	t.Helper()
	return CreateQuestionWithAnswer(t, db, testID, "A")
}

// CreateQuestionWithAnswer inserts a question with a chosen correct answer.
func CreateQuestionWithAnswer(t *testing.T, db *sql.DB, testID int, correct string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO questions (test_id, question_text, option_a, option_b, option_c, option_d,
		correct_answer, marks, neg_marks, importance, difficulty, type, concept_tag, expected_time)
		VALUES ($1, 'Q?', 'a', 'b', 'c', 'd', $2, 4, 1, 'medium', 'M', 'mcq', 'concept-x', 60) RETURNING id`,
		testID, correct).Scan(&id)
	if err != nil {
		t.Fatalf("create question: %v", err)
	}
	return id
}

// Graph is the standard fixture bundle: one tenant with an admin, a coach, a
// subject, a test, a student and an assignment.
type Graph struct {
	TenantID     int
	AdminUserID  int
	CoachUserID  int
	CoachID      int
	SubjectID    int
	TestID       int
	StudentID    int
	AssignmentID int
}

// CreateGraph builds the standard bundle for the calling test. The whole graph
// is removed by the tenant cleanup registered in CreateTenant.
func CreateGraph(t *testing.T, db *sql.DB) *Graph {
	t.Helper()
	g := &Graph{}
	g.TenantID = CreateTenant(t, db)
	g.AdminUserID = CreateAdmin(t, db, g.TenantID)
	g.CoachUserID, g.CoachID = CreateCoach(t, db, g.TenantID)
	g.SubjectID = CreateSubject(t, db, g.TenantID, "Math")
	g.TestID = CreateTest(t, db, g.TenantID, g.SubjectID, g.CoachID, 60)
	g.StudentID = CreateStudent(t, db, g.TenantID, g.CoachID, UniqueCode(t, "stu"), "Fixture Student")
	g.AssignmentID = CreateAssignment(t, db, g.StudentID, g.TestID, g.CoachID)
	return g
}
