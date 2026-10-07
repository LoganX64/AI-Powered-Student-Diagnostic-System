package repository

import (
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestAssignmentCreateGetPolicy(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAssignmentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID := createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID := createStudent(t, db, tenantID, coachID, "sa-1", "S One")

	id, err := r.Create(studentID, testID, coachID, []byte(`{"webcam":false}`), 0, "standard")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	policy, err := r.GetPolicy(id)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	// JSONB normalizes whitespace, so compare by key-value presence, not exact bytes.
	if !strings.Contains(string(policy), `"webcam": false`) {
		t.Fatalf("policy=%s", policy)
	}

	active, err := r.HasActiveAssignment(studentID, testID)
	if err != nil || !active {
		t.Fatalf("HasActiveAssignment: %v %v", active, err)
	}
}

func TestAssignmentReassignmentAllowed(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAssignmentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID := createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID := createStudent(t, db, tenantID, coachID, "sa-2", "S Two")

	if _, err := r.Create(studentID, testID, coachID, []byte(`{}`), 0, "standard"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	// Migration 000018 removed the (student_id, test_id) UNIQUE constraint so a
	// coach can re-assign after submission. The second insert must succeed.
	if _, err := r.Create(studentID, testID, coachID, []byte(`{}`), 0, "standard"); err != nil {
		t.Fatalf("re-assignment must be allowed: %v", err)
	}
}

func TestAssignmentListByStudent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAssignmentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID := createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID := createStudent(t, db, tenantID, coachID, "sa-3", "S Three")
	r.Create(studentID, testID, coachID, []byte(`{}`), 0, "standard")

	list, total, err := r.ListByStudent(studentID, nil, "", 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListByStudent: %v %d %v", err, total, list)
	}
}

func TestAssignmentGetByID(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAssignmentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID := createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID := createStudent(t, db, tenantID, coachID, "sa-4", "S Four")
	id, _ := r.Create(studentID, testID, coachID, []byte(`{}`), 0, "standard")

	asID, teID, status, assignedAt, title, err := r.GetByID(id, studentID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if asID != studentID || teID != testID || title == "" || status == "" || assignedAt == "" {
		t.Fatalf("GetByID=(%d,%d,%q,%q,%q)", asID, teID, status, assignedAt, title)
	}

	if _, _, _, _, _, err := r.GetByID(id, studentID+99999); err == nil {
		t.Fatal("GetByID with wrong student must fail")
	}
}

func TestAssignmentDelete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewAssignmentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Math")
	testID := createTest(t, db, tenantID, subjectID, coachID, 60)
	studentID := createStudent(t, db, tenantID, coachID, "sa-5", "S Five")
	id, _ := r.Create(studentID, testID, coachID, []byte(`{}`), 0, "standard")

	ok, err := r.Delete(id, tenantID, nil)
	if err != nil || !ok {
		t.Fatalf("Delete: %v %v", ok, err)
	}
	if ok, _ := r.HasActiveAssignment(studentID, testID); ok {
		t.Fatal("assignment must be gone after delete")
	}
}
