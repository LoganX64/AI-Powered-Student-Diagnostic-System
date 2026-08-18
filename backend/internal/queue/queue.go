// Package queue provides a pluggable background job queue. In "standard" mode
// it runs an in-process channel; in "scale" mode (Redis enabled) it uses Redis
// Streams so multiple API instances share one durable queue.
package queue

import (
	"ai-student-diagnostic/backend/internal/config"
	"encoding/json"
)

// Task types processed by the queue.
const (
	TaskComputeSQI = "compute_sqi"
	TaskFinalize   = "finalize_submit"
	StreamKey      = "diag:queue"
	ConsumerGroup  = "diag-workers"
)

// AnswerInput is a queue-local copy of the answer shape. Keeping it here avoids
// an import cycle with the services package (which imports queue for the sweeper).
type AnswerInput struct {
	QuestionID        int     `json:"question_id"`
	SelectedAnswer    string  `json:"selected_answer"`
	TimeSpent         float64 `json:"time_spent"`
	Seen              *bool   `json:"seen"`
	MarkedForReview   bool    `json:"marked_for_review"`
	Revisited         bool    `json:"revisited"`
	ChangedAnswer     bool    `json:"changed_answer"`
	WasInitiallyWrong bool    `json:"was_initially_wrong"`
}

// FinalizePayload carries everything needed to finalize a submitted attempt.
// Answers is empty when produced by the sweeper (answers were already autosaved).
type FinalizePayload struct {
	AssignmentID int           `json:"assignment_id"`
	AttemptID    int           `json:"attempt_id"`
	StudentID    int           `json:"student_id"`
	Answers      []AnswerInput `json:"answers"`
}

// ComputePayload carries the job id and its tenant for a compute task. The
// tenant is threaded through so the worker can scope job reads/writes by tenant.
type ComputePayload struct {
	JobID    int `json:"job_id"`
	TenantID int `json:"tenant_id"`
}

// Queue is the job interface used by handlers and the sweeper.
type Queue interface {
	EnqueueCompute(jobID, tenantID int) error
	EnqueueFinalize(p FinalizePayload) error
	Start(computeHandler func(int, int) error, finalizeHandler func(FinalizePayload) error)
	Stop()
}

// New returns the configured queue implementation. Falls back to the
// in-process queue when Redis is unavailable/disabled.
func New(cfg *config.Config) Queue {
	if cfg != nil && cfg.RedisEnabled {
		if client := newRedisClient(cfg); client != nil {
			return &redisQueue{client: client}
		}
	}
	return &inProcessQueue{
		computeCh:  make(chan ComputePayload, 1024),
		finalizeCh: make(chan FinalizePayload, 1024),
	}
}

func marshalCompute(jobID, tenantID int) (string, error) {
	b, err := json.Marshal(ComputePayload{JobID: jobID, TenantID: tenantID})
	return string(b), err
}

func marshalFinalize(p FinalizePayload) (string, error) {
	b, err := json.Marshal(p)
	return string(b), err
}
