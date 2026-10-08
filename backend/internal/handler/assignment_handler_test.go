package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// assignment_handler_test.go tests assignment creation, batch distribution,
// listing, and deletion across AdminHandler and CoachHandler.

func TestCreateAssignmentRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	newStu := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "asgn_s"), "Assign Student")

	t.Run("admin_creates_assignment", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"student_id": %d,
			"test_id": %d,
			"coach_id": %d,
			"integrity_policy": {"video_proctoring": true}
		}`, newStu, f.TestID, f.CoachID))

		f.Admin.CreateAssignment(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			AssignmentID int `json:"assignment_id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.AssignmentID == 0 {
			t.Fatal("expected non-zero assignment_id")
		}

		// Verify in DB
		var sID, tID, cID int
		var status string
		err := f.DB.QueryRow(`SELECT student_id, test_id, coach_id, status FROM assignments WHERE id = $1`,
			resp.AssignmentID).Scan(&sID, &tID, &cID, &status)
		if err != nil {
			t.Fatalf("query assignment: %v", err)
		}
		if sID != newStu || tID != f.TestID || cID != f.CoachID || status != "assigned" {
			t.Fatalf("assignment in DB: student=%d, test=%d, coach=%d, status=%q", sID, tID, cID, status)
		}
	})

	t.Run("rejects_foreign_student", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"student_id": %d,
			"test_id": %d,
			"coach_id": %d
		}`, f.OtherStudentID, f.TestID, f.CoachID))

		f.Admin.CreateAssignment(c)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
			t.Fatalf("code=%d want 4xx error (body=%s)", w.Code, w.Body.String())
		}
	})
}

func TestCreateBatchAssignmentRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	s1 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "bs1"), "Batch Stu 1")
	s2 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "bs2"), "Batch Stu 2")
	batchID := testutil.CreateBatch(t, f.DB, f.TenantID, "Exam Batch")
	_ = f.BatchRepo.SetStudentBatch(f.TenantID, s1, &batchID)
	_ = f.BatchRepo.SetStudentBatch(f.TenantID, s2, &batchID)

	t.Run("admin_creates_batch_assignment", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"test_id": %d,
			"batch_ids": [%d],
			"coach_id": %d
		}`, f.TestID, batchID, f.CoachID))

		f.Admin.CreateBatchAssignment(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Created int `json:"created"`
			Skipped int `json:"skipped"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal batch assignment response: %v", err)
		}
		if resp.Created != 2 {
			t.Fatalf("created=%d want 2", resp.Created)
		}
	})

	t.Run("coach_creates_batch_assignment", func(t *testing.T) {
		s3 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "bs3"), "Batch Stu 3")
		c, w := f.ctxAsCoach(t)
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"test_id": %d,
			"student_ids": [%d]
		}`, f.TestID, s3))

		f.Coach.CreateBatchAssignment(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("coach batch assign code=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestListAndDeleteAssignment(t *testing.T) {
	f := newHandlerFixture(t)
	asgnID := testutil.CreateAssignment(t, f.DB, f.StudentID, f.TestID, f.CoachID)

	// 1. List Assignments
	t.Run("list_assignments", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/assignments?test_id=%d", f.TestID), nil)

		f.Admin.ListAssignments(c)
		if w.Code != http.StatusOK {
			t.Fatalf("list assignments code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Total int `json:"total"`
			Data  []struct {
				ID        int `json:"id"`
				StudentID int `json:"student_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal list: %v", err)
		}
		if resp.Total < 1 {
			t.Fatalf("expected at least 1 assignment, got %d", resp.Total)
		}
	})

	// 2. Admin Delete Assignment
	t.Run("delete_assignment", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(asgnID))

		f.Admin.DeleteAssignment(c)
		if w.Code != http.StatusOK {
			t.Fatalf("delete code=%d body=%s", w.Code, w.Body.String())
		}

		var exists bool
		_ = f.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1)`, asgnID).Scan(&exists)
		if exists {
			t.Fatal("assignment still exists in DB after deletion")
		}
	})
}
