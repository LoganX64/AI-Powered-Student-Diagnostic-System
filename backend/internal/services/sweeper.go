package services

import (
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"context"
	"log"
)

// Sweeper periodically claims expired in-progress attempts and enqueues a
// finalize_submit job so they are finalized (using already-autosaved answers).
// Idempotent via the unique_assignment_attempt constraint + status guard.
type Sweeper struct {
	AttemptRepo    *repository.AttemptRepo
	Queue          queue.Queue
	GraceSeconds   int
}

func NewSweeper(attemptRepo *repository.AttemptRepo, q queue.Queue, graceSeconds int) *Sweeper {
	return &Sweeper{AttemptRepo: attemptRepo, Queue: q, GraceSeconds: graceSeconds}
}

// RunOnce scans for expired attempts and enqueues finalization for each.
func (s *Sweeper) RunOnce(ctx context.Context) {
	expired, err := s.AttemptRepo.ExpiredInProgressAttempts(s.GraceSeconds)
	if err != nil {
		log.Printf("[SWEEPER] scan failed: %v", err)
		return
	}
	for _, e := range expired {
		s.Queue.EnqueueFinalize(queue.FinalizePayload{
			AssignmentID: e.AssignmentID,
			AttemptID:    e.AttemptID,
			Answers:      nil,
		})
	}
}
