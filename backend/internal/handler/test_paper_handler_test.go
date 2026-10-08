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

// test_paper_handler_test.go tests test paper CRUD and question lifecycle:
// CreateTest, UpdateTest, DeleteTest, ListTests, GetTest,
// GetTestQuestions, CreateQuestion, UpdateQuestion, DeleteQuestion.

func TestCreateTestRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)

	t.Run("admin_creates_test_with_coach", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"title": "Geometry Final",
			"subject_id": %d,
			"subject_name": "Geometry",
			"coach_id": %d,
			"duration": 90
		}`, f.SubjectID, f.CoachID))

		f.Admin.CreateTest(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			TestID int `json:"test_id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.TestID == 0 {
			t.Fatal("expected non-zero test_id")
		}

		// Verify in DB
		var title, subjectName string
		var duration, coachID int
		err := f.DB.QueryRow(`SELECT title, subject_name, duration, coach_id FROM tests WHERE id = $1 AND tenant_id = $2`,
			resp.TestID, f.TenantID).Scan(&title, &subjectName, &duration, &coachID)
		if err != nil {
			t.Fatalf("query created test: %v", err)
		}
		if title != "Geometry Final" || subjectName != "Geometry" || duration != 90 || coachID != f.CoachID {
			t.Fatalf("unexpected DB record: title=%q, subject=%q, duration=%d, coach=%d", title, subjectName, duration, coachID)
		}
	})

	t.Run("coach_creates_test_auto_assigns_coach", func(t *testing.T) {
		c, w := f.ctxAsCoach(t)
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"title": "Coach Created Test",
			"subject_id": %d,
			"subject_name": "Math",
			"duration": 45
		}`, f.SubjectID))

		// AdminHandler and CoachHandler share methods or coach can call CreateTest
		f.Admin.CreateTest(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			TestID int `json:"test_id"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		var coachID int
		_ = f.DB.QueryRow(`SELECT coach_id FROM tests WHERE id = $1`, resp.TestID).Scan(&coachID)
		if coachID != f.CoachID {
			t.Fatalf("coach_id=%d want %d", coachID, f.CoachID)
		}
	})

	t.Run("rejects_foreign_coach_id", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{
			"title": "Invalid Coach Test",
			"subject_id": %d,
			"subject_name": "Math",
			"coach_id": %d,
			"duration": 60
		}`, f.SubjectID, f.OtherCoachID))

		f.Admin.CreateTest(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code=%d want 400 (body=%s)", w.Code, w.Body.String())
		}
	})
}

func TestUpdateAndGetTestRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	testID := testutil.CreateTestNamed(t, f.DB, f.TenantID, f.SubjectID, f.CoachID, 60, "Old Title", "Algebra")

	// 1. Update test
	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(testID))
	withJSONBody(c, http.MethodPut, fmt.Sprintf(`{
		"title": "New Title",
		"subject_id": %d,
		"subject_name": "Calculus",
		"coach_id": %d,
		"duration": 120
	}`, f.SubjectID, f.CoachID))

	f.Admin.UpdateTest(c)
	if w.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", w.Code, w.Body.String())
	}

	// 2. Get test detail
	c2, w2 := f.ctx(t, "admin")
	withParam(c2, "id", strconv.Itoa(testID))

	f.Admin.GetTest(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get code=%d body=%s", w2.Code, w2.Body.String())
	}

	var detail struct {
		TestID      int    `json:"test_id"`
		Title       string `json:"title"`
		SubjectName string `json:"subject_name"`
		Duration    int    `json:"duration"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal detail: %v", err)
	}
	if detail.TestID != testID || detail.Title != "New Title" || detail.Duration != 120 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
}

func TestDeleteAndListTests(t *testing.T) {
	f := newHandlerFixture(t)
	testID := testutil.CreateTestNamed(t, f.DB, f.TenantID, f.SubjectID, f.CoachID, 60, "Test To Delete", "Algebra")

	// 1. Verify it appears in ListTests
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodGet, "/tests?search=Test+To+Delete", nil)
	f.Admin.ListTests(c)
	if w.Code != http.StatusOK {
		t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Total int `json:"total"`
		Data  []struct {
			TestID int    `json:"test_id"`
			Title  string `json:"title"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || resp.Data[0].TestID != testID {
		t.Fatalf("expected test in list, got %+v", resp)
	}


	// 2. Soft delete
	c2, w2 := f.ctx(t, "admin")
	withParam(c2, "id", strconv.Itoa(testID))
	f.Admin.DeleteTest(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete code=%d body=%s", w2.Code, w2.Body.String())
	}

	// 3. Hidden from default list
	c3, w3 := f.ctx(t, "admin")
	c3.Request = httptest.NewRequest(http.MethodGet, "/tests?search=Test+To+Delete", nil)
	f.Admin.ListTests(c3)
	_ = json.Unmarshal(w3.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Fatalf("soft-deleted test should be hidden from active list, got total=%d", resp.Total)
	}

	// 4. Visible with include_deleted=true
	c4, w4 := f.ctx(t, "admin")
	c4.Request = httptest.NewRequest(http.MethodGet, "/tests?search=Test+To+Delete&include_deleted=true", nil)
	f.Admin.ListTests(c4)
	_ = json.Unmarshal(w4.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Fatalf("soft-deleted test should be visible with include_deleted=true, got total=%d", resp.Total)
	}
}

func TestQuestionCRUDLifecycle(t *testing.T) {
	f := newHandlerFixture(t)
	testID := testutil.CreateTest(t, f.DB, f.TenantID, f.SubjectID, f.CoachID, 60)

	// 1. Create Question
	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(testID))
	withJSONBody(c, http.MethodPost, `[{
		"question_text": "What is 2 + 2?",
		"option_a": "3",
		"option_b": "4",
		"option_c": "5",
		"option_d": "6",
		"correct_answer": "B",
		"marks": 4,
		"neg_marks": 1,
		"importance": "high",
		"difficulty": "E",
		"type": "mcq",
		"concept_tag": "arithmetic",
		"expected_time": 30
	}]`)

	f.Admin.CreateQuestion(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create question code=%d body=%s", w.Code, w.Body.String())
	}

	var createResp struct {
		QuestionIDs []int `json:"question_ids"`
		Count       int   `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create question: %v", err)
	}
	if len(createResp.QuestionIDs) != 1 {
		t.Fatalf("expected 1 question_id, got %v", createResp.QuestionIDs)
	}
	qid := createResp.QuestionIDs[0]

	// 2. Get Questions
	c2, w2 := f.ctx(t, "admin")
	withParam(c2, "id", strconv.Itoa(testID))
	f.Admin.GetTestQuestions(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get questions code=%d body=%s", w2.Code, w2.Body.String())
	}

	var listQ struct {
		Total int `json:"total"`
		Data  []struct {
			ID           int    `json:"id"`
			QuestionText string `json:"question_text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &listQ); err != nil {
		t.Fatalf("unmarshal questions: %v", err)
	}
	if listQ.Total != 1 || listQ.Data[0].ID != qid || listQ.Data[0].QuestionText != "What is 2 + 2?" {
		t.Fatalf("unexpected questions list: %+v", listQ)
	}

	// 3. Update Question
	c3, w3 := f.ctx(t, "admin")
	withParam(c3, "id", strconv.Itoa(testID))
	c3.Params = append(c3.Params, struct{ Key, Value string }{Key: "qid", Value: strconv.Itoa(qid)})
	withJSONBody(c3, http.MethodPut, `{
		"question_text": "What is 3 + 3?",
		"option_a": "5",
		"option_b": "6",
		"option_c": "7",
		"option_d": "8",
		"correct_answer": "B",
		"marks": 5,
		"neg_marks": 1,
		"importance": "high",
		"difficulty": "M",
		"type": "mcq",
		"concept_tag": "arithmetic",
		"expected_time": 45
	}`)

	f.Admin.UpdateQuestion(c3)
	if w3.Code != http.StatusOK {
		t.Fatalf("update question code=%d body=%s", w3.Code, w3.Body.String())
	}

	var updatedText string
	var updatedMarks float64
	_ = f.DB.QueryRow(`SELECT question_text, marks FROM questions WHERE id = $1`, qid).Scan(&updatedText, &updatedMarks)
	if updatedText != "What is 3 + 3?" || updatedMarks != 5 {
		t.Fatalf("question not updated in DB: text=%q, marks=%v", updatedText, updatedMarks)
	}

	// 4. Delete Question
	c4, w4 := f.ctx(t, "admin")
	withParam(c4, "id", strconv.Itoa(testID))
	c4.Params = append(c4.Params, struct{ Key, Value string }{Key: "qid", Value: strconv.Itoa(qid)})

	f.Admin.DeleteQuestion(c4)
	if w4.Code != http.StatusOK {
		t.Fatalf("delete question code=%d body=%s", w4.Code, w4.Body.String())
	}

	// Verify deleted in DB
	var exists bool
	_ = f.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1)`, qid).Scan(&exists)
	if exists {
		t.Fatal("question still exists in DB after deletion")
	}
}
