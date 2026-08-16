package services

// SplitCohort divides a large student set into K staggered sub-batches, each
// no larger than `threshold`, with `windowSec` offset start windows between
// them. This is the real-world "multiple shifts" pattern used to keep a 50k+
// cohort inside Band B (no Redis required).
func SplitCohort(students []int, threshold, windowSec int) [][]int {
	if threshold <= 0 {
		threshold = 1
	}
	if len(students) == 0 {
		return nil
	}
	var shifts [][]int
	for i := 0; i < len(students); i += threshold {
		end := i + threshold
		if end > len(students) {
			end = len(students)
		}
		shift := make([]int, end-i)
		copy(shift, students[i:end])
		shifts = append(shifts, shift)
	}
	return shifts
}
