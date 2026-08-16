package services

import "testing"

func TestSplitCohort(t *testing.T) {
	students := make([]int, 125)
	for i := range students {
		students[i] = i + 1
	}

	shifts := SplitCohort(students, 50, 1800)
	if len(shifts) != 3 {
		t.Fatalf("expected 3 shifts, got %d", len(shifts))
	}
	if len(shifts[0]) != 50 || len(shifts[1]) != 50 || len(shifts[2]) != 25 {
		t.Fatalf("unexpected shift sizes: %d %d %d", len(shifts[0]), len(shifts[1]), len(shifts[2]))
	}

	// No cohort exceeds the threshold.
	for _, s := range shifts {
		if len(s) > 50 {
			t.Fatalf("shift exceeded threshold: %d", len(s))
		}
	}

	// Empty input → nil.
	if SplitCohort(nil, 50, 0) != nil {
		t.Fatalf("expected nil for empty cohort")
	}
}
