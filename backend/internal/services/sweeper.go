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
	AutosaveBuffer *AutosaveBuffer
	GraceSeconds   int
}

func NewSweeper(attemptRepo *repository.AttemptRepo, q queue.Queue, autosaveBuffer *AutosaveBuffer, graceSeconds int) *Sweeper {
	return &Sweeper{AttemptRepo: attemptRepo, Queue: q, AutosaveBuffer: autosaveBuffer, GraceSeconds: graceSeconds}
}

// RunOnce scans for expired attempts and enqueues finalization for each.
func (s *Sweeper) RunOnce(ctx context.Context) {
	expired, err := s.AttemptRepo.ExpiredInProgressAttempts(s.GraceSeconds)
	if err != nil {
		log.Printf("[SWEEPER] scan failed: %v", err)
		return
	}
	for _, e := range expired {
		// Ensure the latest buffered answers are persisted before finalize,
		// otherwise the finalize (which passes no answers) would commit an
		// incomplete attempt.
		if s.AutosaveBuffer != nil {
			if err := s.AutosaveBuffer.FlushAttempt(e.AttemptID); err != nil {
				log.Printf("[SWEEPER] flush attempt %d before finalize failed: %v", e.AttemptID, err)
			}
		}
		s.Queue.EnqueueFinalize(queue.FinalizePayload{
			AssignmentID: e.AssignmentID,
			AttemptID:    e.AttemptID,
			StudentID:    e.StudentID,
			Answers:      nil,
		})
	}
}
