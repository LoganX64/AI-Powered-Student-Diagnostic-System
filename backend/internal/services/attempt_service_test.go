package services

import (
	"testing"
	"time"
)

// These tests are the regression net for a 60x unit mismatch: tests.duration is
// MINUTES, AnswerInput.TimeSpent is SECONDS. The mismatch made every preset
// unsubmittable â€” the client-timed tier rejected with 400 "total time_spent
// exceeds test duration" after 60s of work on a 60-minute exam, and the
// server-timed tier expired after 60s via the same conversion.

func boolPtr(b bool) *bool { return &b }

func TestValidateAnswers(t *testing.T) {
	correct := map[int]string{1: "A", 2: "B", 3: "C", 4: "D"}

	tests := []struct {
		name      string
		answers   []AnswerInput
		duration  int // minutes
		wantErr   string
		wantNoErr bool
	}{
		{
			name:      "well-formed answers pass",
			answers:   []AnswerInput{{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 30, Seen: boolPtr(true)}, {QuestionID: 2, SelectedAnswer: "B", TimeSpent: 45.5, Seen: boolPtr(true)}},
			duration:  60,
			wantNoErr: true,
		},
		{
			name:    "unknown question id rejected",
			answers: []AnswerInput{{QuestionID: 99, SelectedAnswer: "A", TimeSpent: 5, Seen: boolPtr(true)}},
			wantErr: "invalid question id",
		},
		{
			name:    "duplicate question id rejected",
			answers: []AnswerInput{{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 5, Seen: boolPtr(true)}, {QuestionID: 1, SelectedAnswer: "B", TimeSpent: 5, Seen: boolPtr(true)}},
			wantErr: "duplicate question id",
		},
		{
			name:    "negative time_spent rejected",
			answers: []AnswerInput{{QuestionID: 1, SelectedAnswer: "A", TimeSpent: -1, Seen: boolPtr(true)}},
			wantErr: "time_spent cannot be negative",
		},
		{
			name:    "unseen question cannot carry an answer",
			answers: []AnswerInput{{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 5, Seen: boolPtr(false)}},
			wantErr: "not seen question cannot have selected_answer",
		},
		{
			name:    "option outside A/B/C/D rejected",
			answers: []AnswerInput{{QuestionID: 1, SelectedAnswer: "E", TimeSpent: 5, Seen: boolPtr(true)}},
			wantErr: "selected_answer must be A/B/C/D",
		},
		{
			// The regression: 3601s of work on a 60-minute exam is only 1s over, and
			// used to be rejected. It must be recorded, not refused.
			name: "total time one second over duration is accepted",
			answers: []AnswerInput{
				{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 1800.5, Seen: boolPtr(true)},
				{QuestionID: 2, SelectedAnswer: "B", TimeSpent: 1800.5, Seen: boolPtr(true)},
			},
			duration:  60,
			wantNoErr: true,
		},
		{
			name: "wildly over duration is still accepted",
			answers: []AnswerInput{
				{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 20000, Seen: boolPtr(true)},
			},
			duration:  60,
			wantNoErr: true,
		},
		{
			// Untimed test (duration 0): no ceiling is applied at all.
			name: "zero duration imposes no ceiling",
			answers: []AnswerInput{
				{QuestionID: 1, SelectedAnswer: "A", TimeSpent: 99999, Seen: boolPtr(true)},
			},
			duration:  0,
			wantNoErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAnswers(tc.answers, correct, tc.duration)
			if tc.wantNoErr {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestExamDeadline(t *testing.T) {
	started := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		started  time.Time
		duration int // minutes
		want     time.Time
	}{
		{
			// The regression: 60 minutes must be time.Hour. The old code used
			// time.Second, so a 60-minute exam expired after 60 seconds.
			name:     "60 minutes is one hour",
			started:  started,
			duration: 60,
			want:     started.Add(time.Hour),
		},
		{
			name:     "45 minutes",
			started:  started,
			duration: 45,
			want:     started.Add(45 * time.Minute),
		},
		{
			name:     "1 minute",
			started:  started,
			duration: 1,
			want:     started.Add(time.Minute),
		},
		{
			name:     "zero duration means untimed",
			started:  started,
			duration: 0,
			want:     time.Time{},
		},
		{
			name:     "negative duration means untimed",
			started:  started,
			duration: -5,
			want:     time.Time{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExamDeadline(tc.started, tc.duration)
			if !got.Equal(tc.want) {
				t.Fatalf("ExamDeadline(%v, %d) = %v, want %v", tc.started, tc.duration, got, tc.want)
			}
		})
	}
}
