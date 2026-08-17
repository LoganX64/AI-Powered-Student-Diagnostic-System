package queue

import "testing"

// Validates the in-process queue dispatches both task types to their handlers.
func TestInProcessQueue(t *testing.T) {
	q := &inProcessQueue{
		computeCh:  make(chan ComputePayload, 4),
		finalizeCh: make(chan FinalizePayload, 4),
	}

	gotCompute := make(chan int, 1)
	gotFinalize := make(chan FinalizePayload, 1)
	q.Start(func(jobID, tenantID int) { gotCompute <- jobID }, func(p FinalizePayload) { gotFinalize <- p })

	q.EnqueueCompute(42, 1)
	q.EnqueueFinalize(FinalizePayload{AssignmentID: 7, AttemptID: 8})

	if id := <-gotCompute; id != 42 {
		t.Fatalf("expected compute job 42, got %d", id)
	}
	p := <-gotFinalize
	if p.AssignmentID != 7 || p.AttemptID != 8 {
		t.Fatalf("unexpected finalize payload: %+v", p)
	}
}
