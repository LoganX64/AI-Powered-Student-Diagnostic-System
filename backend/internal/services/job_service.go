package services

import (
	"encoding/json"
	"log"

	"ai-student-diagnostic/backend/internal/repository"
)

type JobService struct {
	JobRepo             *repository.JobRepo
	AttemptService      *AttemptService
	ChunkSize           int
	NotificationService *NotificationService
}

func NewJobService(jobRepo *repository.JobRepo, attemptService *AttemptService, chunkSize int, notifService *NotificationService) *JobService {
	return &JobService{JobRepo: jobRepo, AttemptService: attemptService, ChunkSize: chunkSize, NotificationService: notifService}
}

type computeJobPayload struct {
	AttemptIDs []int `json:"attempt_ids"`
}

// chunkSize returns the configured chunk size, falling back to 100 when unset.
func (s *JobService) chunkSize() int {
	if s.ChunkSize <= 0 {
		return 100
	}
	return s.ChunkSize
}

// terminalStatus maps final progress counters onto a terminal job status.
// No failures means completed; no successes means failed; a mix is partial.
func terminalStatus(done, failed int) string {
	switch {
	case failed == 0:
		return "completed"
	case done == 0:
		return "failed"
	default:
		return "partial"
	}
}

// Process runs a single job (currently only compute_sqi). It streams progress
// into the jobs table so clients can poll done/total. A job never aborts on a
// single attempt failure — failures are counted and the run continues.
// It returns an error only for transient failures (e.g. the job row or its
// status cannot be read/written) so the queue can leave the message
// unacknowledged and replay it (at-least-once). An unparseable payload or an
// already-finished job is treated as terminal (nil error) so it is acked.
func (s *JobService) Process(jobID, tenantID int) error {
	job, err := s.JobRepo.Get(jobID, tenantID)
	if err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "partial" {
		return nil
	}
	if err := s.JobRepo.SetStatus(jobID, tenantID, "running"); err != nil {
		return err
	}

	var pld computeJobPayload
	if err := json.Unmarshal(job.Payload, &pld); err != nil {
		if err := s.JobRepo.SetStatus(jobID, tenantID, "failed"); err != nil {
			log.Printf("[JOB] set status failed for job %d: %v", jobID, err)
		}
		return nil
	}

	total := len(pld.AttemptIDs)
	if total == 0 {
		if err := s.JobRepo.SetStatus(jobID, tenantID, "completed"); err != nil {
			log.Printf("[JOB] set status failed for job %d: %v", jobID, err)
		}
		if s.NotificationService != nil {
			if err := s.NotificationService.NotifySQIComplete(tenantID, jobID, 0, 0); err != nil {
				log.Printf("[JOB] notify SQI complete failed for job %d: %v", jobID, err)
			}
		}
		return nil
	}

	chunk := s.chunkSize()

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
				if ierr := s.JobRepo.Increment(jobID, tenantID, 0, 1); ierr != nil {
					log.Printf("[JOB] increment failed for job %d: %v", jobID, ierr)
				}
				failed++
				continue
			}
			if err := s.AttemptService.ComputeAndStoreAttempt(attemptID, testID); err != nil {
				if ierr := s.JobRepo.Increment(jobID, tenantID, 0, 1); ierr != nil {
					log.Printf("[JOB] increment failed for job %d: %v", jobID, ierr)
				}
				failed++
				continue
			}
			if ierr := s.JobRepo.Increment(jobID, tenantID, 1, 0); ierr != nil {
				log.Printf("[JOB] increment failed for job %d: %v", jobID, ierr)
			}
			done++
		}
	}

	switch terminalStatus(done, failed) {
	case "completed":
		if err := s.JobRepo.SetStatus(jobID, tenantID, "completed"); err != nil {
			log.Printf("[JOB] set status failed for job %d: %v", jobID, err)
		}
		if s.NotificationService != nil {
			if err := s.NotificationService.NotifySQIComplete(tenantID, jobID, done, failed); err != nil {
				log.Printf("[JOB] notify SQI complete failed for job %d: %v", jobID, err)
			}
		}
	case "failed":
		if err := s.JobRepo.SetStatus(jobID, tenantID, "failed"); err != nil {
			log.Printf("[JOB] set status failed for job %d: %v", jobID, err)
		}
	default:
		if err := s.JobRepo.SetStatus(jobID, tenantID, "partial"); err != nil {
			log.Printf("[JOB] set status failed for job %d: %v", jobID, err)
		}
	}
	return nil
}

// TestIDForAttempt is a thin wrapper so the worker can resolve a test for an attempt.
func (s *AttemptService) TestIDForAttempt(attemptID int) (int, error) {
	return s.AttemptRepo.GetTestIDForAttempt(attemptID)
}
