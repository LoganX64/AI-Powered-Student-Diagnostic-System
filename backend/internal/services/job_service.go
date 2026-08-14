package services

import (
	"encoding/json"

	"ai-student-diagnostic/backend/internal/repository"
)

type JobService struct {
	JobRepo        *repository.JobRepo
	AttemptService *AttemptService
	ChunkSize      int
}

func NewJobService(jobRepo *repository.JobRepo, attemptService *AttemptService, chunkSize int) *JobService {
	return &JobService{JobRepo: jobRepo, AttemptService: attemptService, ChunkSize: chunkSize}
}

type computeJobPayload struct {
	AttemptIDs []int `json:"attempt_ids"`
}

// Process runs a single job (currently only compute_sqi). It streams progress
// into the jobs table so clients can poll done/total. A job never aborts on a
// single attempt failure — failures are counted and the run continues.
func (s *JobService) Process(jobID int) {
	job, err := s.JobRepo.GetByID(jobID)
	if err != nil {
		return
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "partial" {
		return
	}
	_ = s.JobRepo.SetStatus(jobID, "running")

	var pld computeJobPayload
	if err := json.Unmarshal(job.Payload, &pld); err != nil {
		_ = s.JobRepo.SetStatus(jobID, "failed")
		return
	}

	total := len(pld.AttemptIDs)
	if total == 0 {
		_ = s.JobRepo.SetStatus(jobID, "completed")
		return
	}

	chunk := s.ChunkSize
	if chunk <= 0 {
		chunk = 100
	}

	done := 0
	failed := 0
	for i := 0; i < total; i += chunk {
		end := i + chunk
		if end > total {
			end = total
		}
		for _, attemptID := range pld.AttemptIDs[i:end] {
			testID, err := s.AttemptService.TestIDForAttempt(attemptID)
			if err != nil {
				_ = s.JobRepo.Increment(jobID, 0, 1)
				failed++
				continue
			}
			if err := s.AttemptService.ComputeAndStoreAttempt(attemptID, testID); err != nil {
				_ = s.JobRepo.Increment(jobID, 0, 1)
				failed++
				continue
			}
			_ = s.JobRepo.Increment(jobID, 1, 0)
			done++
		}
	}

	switch {
	case failed == 0:
		_ = s.JobRepo.SetStatus(jobID, "completed")
	case done == 0:
		_ = s.JobRepo.SetStatus(jobID, "failed")
	default:
		_ = s.JobRepo.SetStatus(jobID, "partial")
	}
}

// TestIDForAttempt is a thin wrapper so the worker can resolve a test for an attempt.
func (s *AttemptService) TestIDForAttempt(attemptID int) (int, error) {
	return s.AttemptRepo.GetTestIDForAttempt(attemptID)
}
