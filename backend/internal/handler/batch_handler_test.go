package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// batch_handler_test.go tests batch management and student membership:
// CreateBatch, ListBatches, UpdateBatch, DeleteBatch, TransferStudentBatch,
// and batch expansion.

func TestBatchCRUDLifecycle(t *testing.T) {
	f := newHandlerFixture(t)

	// 1. Admin creates a batch
	var batchID int
	t.Run("admin_creates_batch", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, `{"name":"Alpha Cohort"}`)

		f.Admin.CreateBatch(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			BatchID int `json:"batch_id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.BatchID == 0 {
			t.Fatal("expected non-zero batch_id")
		}
		batchID = resp.BatchID

		// Verify name in DB
		var name string
		_ = f.DB.QueryRow(`SELECT name FROM batches WHERE id = $1 AND tenant_id = $2`, batchID, f.TenantID).Scan(&name)
		if name != "Alpha Cohort" {
			t.Fatalf("name=%q want 'Alpha Cohort'", name)
		}
	})

	// 2. Coach creates a batch
	t.Run("coach_creates_batch", func(t *testing.T) {
		c, w := f.ctxAsCoach(t)
		withJSONBody(c, http.MethodPost, `{"name":"Coach Cohort"}`)

		f.Coach.CreateBatch(c)
		if w.Code != http.StatusCreated {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	})

	// 3. Update batch name
	t.Run("update_batch", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(batchID))
		withJSONBody(c, http.MethodPut, `{"name":"Alpha Cohort Updated"}`)

		f.Admin.UpdateBatch(c)
		if w.Code != http.StatusOK {
			t.Fatalf("update code=%d body=%s", w.Code, w.Body.String())
		}

		var name string
		_ = f.DB.QueryRow(`SELECT name FROM batches WHERE id = $1`, batchID).Scan(&name)
		if name != "Alpha Cohort Updated" {
			t.Fatalf("name=%q want 'Alpha Cohort Updated'", name)
		}
	})

	// 4. List batches and verify
	t.Run("list_batches", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		f.Admin.ListBatches(c)
		if w.Code != http.StatusOK {
			t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Data []struct {
				ID           int    `json:"id"`
				Name         string `json:"name"`
				StudentCount int    `json:"student_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal list: %v", err)
		}
		if len(resp.Data) < 2 {
			t.Fatalf("expected at least 2 batches, got %d", len(resp.Data))
		}
	})

	// 5. Transfer Student into batch and clear
	t.Run("transfer_student_batch", func(t *testing.T) {
		stuID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "b_stu"), "Batch Student")

		// Assign into batch
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(stuID))
		withJSONBody(c, http.MethodPut, fmt.Sprintf(`{"batch_id":%d}`, batchID))

		f.Admin.TransferStudentBatch(c)
		if w.Code != http.StatusOK {
			t.Fatalf("transfer code=%d body=%s", w.Code, w.Body.String())
		}

		var currentBatch sql.NullInt64
		_ = f.DB.QueryRow(`SELECT batch_id FROM students WHERE id = $1`, stuID).Scan(&currentBatch)
		if !currentBatch.Valid || int(currentBatch.Int64) != batchID {
			t.Fatalf("student batch_id=%v want %d", currentBatch, batchID)
		}

		// Clear batch (batch_id = null)
		c2, w2 := f.ctx(t, "admin")
		withParam(c2, "id", strconv.Itoa(stuID))
		withJSONBody(c2, http.MethodPut, `{"batch_id":null}`)

		f.Admin.TransferStudentBatch(c2)
		if w2.Code != http.StatusOK {
			t.Fatalf("clear batch code=%d body=%s", w2.Code, w2.Body.String())
		}

		_ = f.DB.QueryRow(`SELECT batch_id FROM students WHERE id = $1`, stuID).Scan(&currentBatch)
		if currentBatch.Valid {
			t.Fatalf("student batch_id should be NULL, got %v", currentBatch)
		}
	})

	// 6. Delete Batch
	t.Run("delete_batch", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withParam(c, "id", strconv.Itoa(batchID))

		f.Admin.DeleteBatch(c)
		if w.Code != http.StatusOK {
			t.Fatalf("delete code=%d body=%s", w.Code, w.Body.String())
		}

		var exists bool
		_ = f.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM batches WHERE id = $1)`, batchID).Scan(&exists)
		if exists {
			t.Fatal("batch still exists in DB after deletion")
		}
	})
}

func TestExpandBatchTargetsDeduplication(t *testing.T) {
	f := newHandlerFixture(t)
	b1 := testutil.CreateBatch(t, f.DB, f.TenantID, "Batch 1")
	s1 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "exp1"), "Exp 1")
	s2 := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "exp2"), "Exp 2")

	_ = f.BatchRepo.SetStudentBatch(f.TenantID, s1, &b1)
	_ = f.BatchRepo.SetStudentBatch(f.TenantID, s2, &b1)

	c, _ := f.ctx(t, "admin")
	// Supply s1 explicitly AND via b1; s1 should appear only once
	targets, err := f.Admin.expandBatchTargets(c, f.TenantID, []int{s1}, []int{b1})
	if err != nil {
		t.Fatalf("expandBatchTargets: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("expected 2 unique targets, got %d (%v)", len(targets), targets)
	}
}
