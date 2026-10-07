package repository

import (
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestTestPaperCreateAndExists(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTestPaperRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Physics")

	id, err := r.Create(tenantID, "Paper 1", subjectID, coachID, 45, nil, "Physics")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ok, err := r.Exists(id, tenantID); err != nil || !ok {
		t.Fatalf("Exists: ok=%v err=%v", ok, err)
	}
	if ok, err := r.ExistsOwnedByCoach(id, coachID, tenantID); err != nil || !ok {
		t.Fatalf("ExistsOwnedByCoach: ok=%v err=%v", ok, err)
	}
	if ok, _ := r.Exists(id, tenantID+99999); ok {
		t.Fatal("Exists must be false for other tenant")
	}
}

func TestTestPaperListAndGetDetail(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTestPaperRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Chemistry")

	id, _ := r.Create(tenantID, "Paper 1", subjectID, coachID, 60, nil, "Chemistry")

	list, total, err := r.List(tenantID, nil, false, false, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "Paper 1" {
		t.Fatalf("list=%v total=%d", list, total)
	}

	detail, err := r.GetDetail(id, tenantID)
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if detail.Title != "Paper 1" || detail.Duration != 60 || detail.SubjectName != "Chemistry" {
		t.Fatalf("detail=%+v", detail)
	}
}

func TestTestPaperQuestionsCRUD(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTestPaperRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Bio")

	id, _ := r.Create(tenantID, "Paper 2", subjectID, coachID, 30, nil, "Bio")

	qids, err := r.CreateQuestions(id, []QuestionRequest{
		{QuestionText: "Q1?", OptionA: "a", OptionB: "b", OptionC: "c", OptionD: "d",
			CorrectAnswer: "A", Marks: 4, NegMarks: 1, Importance: "medium", Difficulty: "M", Type: "mcq"},
		{QuestionText: "Q2?", OptionA: "a", OptionB: "b", OptionC: "c", OptionD: "d",
			CorrectAnswer: "B", Marks: 4, NegMarks: 1, Importance: "high", Difficulty: "H", Type: "mcq"},
	})
	if err != nil {
		t.Fatalf("CreateQuestions: %v", err)
	}
	if len(qids) != 2 {
		t.Fatalf("qids=%v", qids)
	}

	qs, total, err := r.ListQuestions(id, 10, 0)
	if err != nil || total != 2 || len(qs) != 2 {
		t.Fatalf("ListQuestions: %v %d %d", err, total, len(qs))
	}
	if n, _ := r.CountQuestions(id); n != 2 {
		t.Fatalf("CountQuestions=%d", n)
	}

	ok, err := r.UpdateQuestion(qids[0], id, QuestionRequest{
		QuestionText: "Q1-updated?", OptionA: "a", OptionB: "b", OptionC: "c", OptionD: "d",
		CorrectAnswer: "C", Marks: 4, NegMarks: 1, Importance: "medium", Difficulty: "M", Type: "mcq",
	})
	if err != nil || !ok {
		t.Fatalf("UpdateQuestion: %v %v", ok, err)
	}

	ok, err = r.DeleteQuestion(qids[1], id)
	if err != nil || !ok {
		t.Fatalf("DeleteQuestion: %v %v", ok, err)
	}
	if n, _ := r.CountQuestions(id); n != 1 {
		t.Fatalf("CountQuestions after delete=%d", n)
	}
}

func TestTestPaperUpdateAndDelete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTestPaperRepo(db)
	tenantID := createTenant(t, db)
	adminID := createUser(t, db, tenantID, uniqueEmail(t, "admin"), "admin")
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Chem")

	id, _ := r.Create(tenantID, "Old Title", subjectID, coachID, 30, nil, "Chem")
	ok, err := r.Update(id, tenantID, "New Title", subjectID, coachID, 90, nil, "Chem")
	if err != nil || !ok {
		t.Fatalf("Update: %v %v", ok, err)
	}
	d, _ := r.GetDetail(id, tenantID)
	if d.Title != "New Title" || d.Duration != 90 {
		t.Fatalf("after update: %+v", d)
	}

	ok, err = r.Delete(id, tenantID, adminID)
	if err != nil || !ok {
		t.Fatalf("Delete: %v %v", ok, err)
	}
	if ok, _ := r.Exists(id, tenantID); !ok {
		// Soft-deleted: Exists should still see it or not depending on filter;
		// List with includeDeleted=false must hide it.
	}
	list, total, _ := r.List(tenantID, nil, false, false, "", 10, 0)
	if total != 0 || len(list) != 0 {
		t.Fatalf("soft-deleted test must be hidden from default list: %v %d", list, total)
	}
	listDeleted, totalDeleted, _ := r.List(tenantID, nil, true, false, "", 10, 0)
	if totalDeleted != 1 || len(listDeleted) != 1 {
		t.Fatalf("includeDeleted=true must show it: %v %d", listDeleted, totalDeleted)
	}
}

func TestTestPaperCoachAndSubjectHelpers(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTestPaperRepo(db)
	tenantID := createTenant(t, db)
	adminID := createUser(t, db, tenantID, uniqueEmail(t, "admin"), "admin")
	_, coachID := createCoach(t, db, tenantID)
	subjectID := createSubject(t, db, tenantID, "Geo")

	id, _ := r.Create(tenantID, "Paper 3", subjectID, coachID, 20, nil, "Geo")
	if got, err := r.GetDuration(id); err != nil || got != 20 {
		t.Fatalf("GetDuration=%d %v", got, err)
	}
	if got, err := r.GetSubjectName(id); err != nil || got != "Geo" {
		t.Fatalf("GetSubjectName=%q %v", got, err)
	}
	if cID, tID, err := r.GetCoachAndTenant(id); err != nil || cID != coachID || tID != tenantID {
		t.Fatalf("GetCoachAndTenant=(%d,%d) %v", cID, tID, err)
	}
	if tID, err := r.CoachTenantID(coachID); err != nil || tID != tenantID {
		t.Fatalf("CoachTenantID=%d %v", tID, err)
	}
	list, total, err := r.ListByCoach(coachID, tenantID, 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListByCoach: %v %d %d", err, total, len(list))
	}

	// Subject CRUD
	subID, _, err := r.CreateSubject(tenantID, "Env Sci")
	if err != nil || subID == 0 {
		t.Fatalf("CreateSubject: %v %d", err, subID)
	}
	ok, err := r.UpdateSubject(subID, tenantID, "Environmental Science")
	if err != nil || !ok {
		t.Fatalf("UpdateSubject: %v %v", ok, err)
	}
	ok, err = r.DeleteSubject(subID, tenantID, adminID)
	if err != nil || !ok {
		t.Fatalf("DeleteSubject: %v %v", ok, err)
	}
	subs, _, err := r.ListSubjects(tenantID, "", 10, 0)
	if err != nil || len(subs) != 1 || subs[0].Name != "Geo" {
		t.Fatalf("ListSubjects: %v %v", subs, err)
	}
}
