package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// student_ops_test.go covers the student lifecycle across AdminHandler and CoachHandler:
// CreateStudent, UpdateStudent, ListStudents, GetStudent, DeleteStudent,
// ReactivateStudent, ListStudentAssignments, GetStudentSQI, GetStudentSQIBatch,
// and GetAssignmentResults.

func studentDeletedAt(t *testing.T, f *handlerFixture, studentID, tenantID int) sql.NullTime {
	t.Helper()
	var deleted sql.NullTime
	if err := f.DB.QueryRow(`SELECT deleted_at FROM students WHERE id = $1 AND tenant_id = $2`,
		studentID, tenantID).Scan(&deleted); err != nil {
		t.Fatalf("read student %d deleted_at: %v", studentID, err)
	}
	return deleted
}

func TestAdminCreateStudentRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	batchID := testutil.CreateBatch(t, f.DB, f.TenantID, "Alpha Batch")

	t.Run("auto_generated_student_code", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"name":"Alice Auto","coach_id":%d,"batch_id":%d}`, f.CoachID, batchID))

		f.Admin.CreateStudent(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			StudentID   int    `json:"student_id"`
			StudentCode string `json:"student_code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.StudentID == 0 || resp.StudentCode == "" {
			t.Fatalf("unexpected student_id=%d, student_code=%q", resp.StudentID, resp.StudentCode)
		}

		// Verify in DB
		var name string
		var gotCoachID int
		var gotBatchID sql.NullInt64
		err := f.DB.QueryRow(`SELECT name, coach_id, batch_id FROM students WHERE id = $1 AND tenant_id = $2`,
			resp.StudentID, f.TenantID).Scan(&name, &gotCoachID, &gotBatchID)
		if err != nil {
			t.Fatalf("read created student: %v", err)
		}
		if name != "Alice Auto" || gotCoachID != f.CoachID || !gotBatchID.Valid || int(gotBatchID.Int64) != batchID {
			t.Fatalf("student in DB: name=%q, coach=%d, batch=%v", name, gotCoachID, gotBatchID)
		}
	})

	t.Run("explicit_student_code", func(t *testing.T) {
		explicitCode := testutil.UniqueCode(t, "exp")
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"name":"Bob Explicit","student_code":%q,"coach_id":%d}`, explicitCode, f.CoachID))

		f.Admin.CreateStudent(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			StudentID   int    `json:"student_id"`
			StudentCode string `json:"student_code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.StudentCode != explicitCode {
			t.Fatalf("student_code=%q want %q", resp.StudentCode, explicitCode)
		}
	})
}

func TestAdminCreateStudentRejections(t *testing.T) {
	f := newHandlerFixture(t)

	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "empty_name",
			body:     fmt.Sprintf(`{"name":"","coach_id":%d}`, f.CoachID),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "foreign_coach_id",
			body:     fmt.Sprintf(`{"name":"Bad Coach","coach_id":%d}`, f.OtherCoachID),
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := f.ctx(t, "admin")
			withJSONBody(c, http.MethodPost, tc.body)

			f.Admin.CreateStudent(c)
			if w.Code != tc.wantCode {
				t.Fatalf("code=%d want %d (body=%s)", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestCoachCreateStudent(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctxAsCoach(t)
	withJSONBody(c, http.MethodPost, `{"name":"Coach's Student"}`)

	f.Coach.CreateStudent(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		StudentID   int    `json:"student_id"`
		StudentCode string `json:"student_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// Must be assigned to the calling coach
	var gotCoachID int
	if err := f.DB.QueryRow(`SELECT coach_id FROM students WHERE id = $1`, resp.StudentID).Scan(&gotCoachID); err != nil {
		t.Fatalf("query student coach: %v", err)
	}
	if gotCoachID != f.CoachID {
		t.Fatalf("coach_id=%d want %d", gotCoachID, f.CoachID)
	}
}

func TestUpdateStudentRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "orig"), "Original Name")
	batchID := testutil.CreateBatch(t, f.DB, f.TenantID, "New Batch")
	_, secondCoachID := testutil.CreateCoach(t, f.DB, f.TenantID)

	t.Run("admin_updates_all_fields", func(t *testing.T) {
		newCode := testutil.UniqueCode(t, "upd")
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(studentID))
		withJSONBody(c, http.MethodPut, fmt.Sprintf(`{"name":"Updated Name","student_code":%q,"coach_id":%d,"batch_id":%d}`,
			newCode, secondCoachID, batchID))

		f.Admin.UpdateStudent(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		// Verify state in DB
		var name, code string
		var coach int
		var bID sql.NullInt64
		err := f.DB.QueryRow(`SELECT name, student_code, coach_id, batch_id FROM students WHERE id = $1`, studentID).
			Scan(&name, &code, &coach, &bID)
		if err != nil {
			t.Fatalf("read updated student: %v", err)
		}
		if name != "Updated Name" || code != newCode || coach != secondCoachID || !bID.Valid || int(bID.Int64) != batchID {
			t.Fatalf("student state: name=%q, code=%q, coach=%d, batch=%v", name, code, coach, bID)
		}
	})

	t.Run("rejects_duplicate_student_code_on_update", func(t *testing.T) {
		takenCode := testutil.UniqueCode(t, "taken")
		_ = testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, takenCode, "Other Stu")

		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(studentID))
		withJSONBody(c, http.MethodPut, fmt.Sprintf(`{"name":"Attempted Dup","student_code":%q,"coach_id":%d}`, takenCode, f.CoachID))

		f.Admin.UpdateStudent(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code=%d want 400 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("coach_updates_own_student", func(t *testing.T) {
		coachStuID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "cstu"), "Coach Stu")
		c, w := f.ctxAsCoach(t)
		withParam(c, "id", strconv.Itoa(coachStuID))
		withJSONBody(c, http.MethodPut, `{"name":"Coach Updated Stu"}`)

		f.Coach.UpdateStudent(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var name string
		_ = f.DB.QueryRow(`SELECT name FROM students WHERE id = $1`, coachStuID).Scan(&name)
		if name != "Coach Updated Stu" {
			t.Fatalf("name=%q want 'Coach Updated Stu'", name)
		}
	})
}

func TestAdminListAndGetStudents(t *testing.T) {
	f := newHandlerFixture(t)
	b1 := testutil.CreateBatch(t, f.DB, f.TenantID, "Batch 1")
	s1 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "l1"), "Searchable Alpha")
	_ = f.BatchRepo.SetStudentBatch(f.TenantID, s1, &b1)

	t.Run("list_with_search_and_batch_filter", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/students?search=Searchable&batch_id=%d", b1), nil)

		f.Admin.ListStudents(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Total int `json:"total"`
			Data  []struct {
				StudentID int    `json:"student_id"`
				Name      string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal list: %v", err)
		}
		if resp.Total != 1 || len(resp.Data) != 1 || resp.Data[0].StudentID != s1 {
			t.Fatalf("unexpected list output: %+v", resp)
		}
	})

	t.Run("get_student_detail", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(s1))

		f.Admin.GetStudent(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var detail struct {
			StudentID int    `json:"student_id"`
			Name      string `json:"name"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
			t.Fatalf("unmarshal detail: %v", err)
		}
		if detail.StudentID != s1 || detail.Name != "Searchable Alpha" {
			t.Fatalf("detail=%+v", detail)
		}
	})
}

func TestDeleteAndReactivateStudentRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	stuID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "del"), "To Be Deleted")

	// 1. Soft delete
	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(stuID))

	f.Admin.DeleteStudent(c)
	if w.Code != http.StatusOK {
		t.Fatalf("delete code=%d body=%s", w.Code, w.Body.String())
	}

	del := studentDeletedAt(t, f, stuID, f.TenantID)
	if !del.Valid {
		t.Fatal("student deleted_at should be set after soft delete")
	}

	// 2. Hidden from default active listing
	c2, w2 := f.ctx(t, "admin")
	c2.Request = httptest.NewRequest(http.MethodGet, "/students?search=To+Be+Deleted", nil)
	f.Admin.ListStudents(c2)
	var resp struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Fatalf("soft-deleted student should be hidden from default list, got total=%d", resp.Total)
	}

	// 3. Appears with include_deactivated=true
	c3, w3 := f.ctx(t, "admin")
	c3.Request = httptest.NewRequest(http.MethodGet, "/students?search=To+Be+Deleted&include_deactivated=true", nil)
	f.Admin.ListStudents(c3)
	_ = json.Unmarshal(w3.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Fatalf("soft-deleted student should be visible with include_deactivated=true, got total=%d", resp.Total)
	}

	// 4. Reactivate
	c4, w4 := f.ctx(t, "admin")
	withParam(c4, "id", strconv.Itoa(stuID))
	f.Admin.ReactivateStudent(c4)
	if w4.Code != http.StatusOK {
		t.Fatalf("reactivate code=%d body=%s", w4.Code, w4.Body.String())
	}

	del2 := studentDeletedAt(t, f, stuID, f.TenantID)
	if del2.Valid {
		t.Fatal("student deleted_at should be NULL after reactivation")
	}
}

func TestListStudentAssignments(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.StudentID))

	f.Admin.ListStudentAssignments(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Total int `json:"total"`
		Data  []struct {
			ID     int    `json:"id"`
			TestID int    `json:"test_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal assignments: %v", err)
	}
	if resp.Total < 1 || len(resp.Data) < 1 || resp.Data[0].ID != f.AssignmentID {
		t.Fatalf("unexpected assignments output: %+v", resp)
	}
}

func TestGetStudentSQIAndBatch(t *testing.T) {
	f := newHandlerFixture(t)

	t.Run("single_student_sqi", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(f.StudentID))

		f.Admin.GetStudentSQI(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("batch_sqi_happy_path", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"student_ids":[%d]}`, f.StudentID))

		f.Admin.GetStudentSQIBatch(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("batch_sqi_empty_rejected", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, `{"student_ids":[]}`)

		f.Admin.GetStudentSQIBatch(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code=%d want 400", w.Code)
		}
	})
}

func TestGetAssignmentResults(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.StudentID))
	c.Params = append(c.Params, struct{ Key, Value string }{Key: "assignmentId", Value: strconv.Itoa(f.AssignmentID)})

	f.Admin.GetAssignmentResults(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Student struct {
			ID int `json:"id"`
		} `json:"student"`
		Assignment struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"assignment"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal results: %v", err)
	}
	if resp.Student.ID != f.StudentID || resp.Assignment.ID != f.AssignmentID {
		t.Fatalf("results mismatch: %+v", resp)
	}
}

