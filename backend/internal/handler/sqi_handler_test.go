package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

// sqi_handler_test.go tests diagnostic job creation, batch queuing,
// and job polling across AdminHandler and CoachHandler.

func createAttemptRow(t *testing.T, f *handlerFixture, assignmentID int) int {
	t.Helper()
	var attemptID int
	err := f.DB.QueryRow(`INSERT INTO attempts (assignment_id, status, submitted_at) VALUES ($1, 'submitted', NOW()) RETURNING id`,
		assignmentID).Scan(&attemptID)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	return attemptID
}

func TestComputeSQIRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	attemptID := createAttemptRow(t, f, f.AssignmentID)

	t.Run("admin_computes_single_attempt", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"attempt_id":%d}`, attemptID))

		f.Admin.ComputeSQI(c)
		if w.Code != http.StatusAccepted {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			JobID int `json:"job_id"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.JobID == 0 || resp.Total != 1 {
			t.Fatalf("unexpected response: %+v", resp)
		}

		// Verify job in DB
		var status, jobType string
		err := f.DB.QueryRow(`SELECT status, type FROM jobs WHERE id = $1 AND tenant_id = $2`,
			resp.JobID, f.TenantID).Scan(&status, &jobType)
		if err != nil {
			t.Fatalf("query job: %v", err)
		}
		if status != "pending" || jobType != "compute_sqi" {
			t.Fatalf("job in DB: status=%q, type=%q", status, jobType)
		}

		// 2. Poll GetJob
		c2, w2 := f.ctx(t, "admin")
		withParam(c2, "id", strconv.Itoa(resp.JobID))

		f.Admin.GetJob(c2)
		if w2.Code != http.StatusOK {
			t.Fatalf("get job code=%d body=%s", w2.Code, w2.Body.String())
		}

		var jobDetail struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &jobDetail); err != nil {
			t.Fatalf("unmarshal job detail: %v", err)
		}
		if jobDetail.ID != resp.JobID || jobDetail.Status != "pending" {
			t.Fatalf("job detail mismatch: %+v", jobDetail)
		}
	})

	t.Run("coach_computes_single_attempt", func(t *testing.T) {
		c, w := f.ctxAsCoach(t)
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"attempt_id":%d}`, attemptID))

		f.Coach.ComputeSQI(c)
		if w.Code != http.StatusAccepted {
			t.Fatalf("coach compute code=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestComputeSQIBatchRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	att1 := createAttemptRow(t, f, f.AssignmentID)

	t.Run("admin_computes_batch_by_attempt_ids", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"attempt_ids":[%d]}`, att1))

		f.Admin.ComputeSQIBatch(c)
		if w.Code != http.StatusAccepted {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			JobID int `json:"job_id"`
			Total int `json:"total"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.JobID == 0 || resp.Total != 1 {
			t.Fatalf("unexpected batch response: %+v", resp)
		}
	})

	t.Run("admin_computes_batch_by_test_id", func(t *testing.T) {
		c, w := f.ctx(t, "admin")
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"test_id":%d}`, f.TestID))

		f.Admin.ComputeSQIBatch(c)
		if w.Code != http.StatusAccepted {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			JobID int `json:"job_id"`
			Total int `json:"total"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.JobID == 0 || resp.Total < 1 {
			t.Fatalf("unexpected test_id batch response: %+v", resp)
		}
	})

	t.Run("coach_computes_batch", func(t *testing.T) {
		c, w := f.ctxAsCoach(t)
		withJSONBody(c, http.MethodPost, fmt.Sprintf(`{"attempt_ids":[%d]}`, att1))

		f.Coach.ComputeSQIBatch(c)
		if w.Code != http.StatusAccepted {
			t.Fatalf("coach compute batch code=%d body=%s", w.Code, w.Body.String())
		}
	})
}
