package types

// ─────────────────────────────────────────────
// DIAGNOSTIC OUTPUT TYPES
// Shared between handlers and services.
// ─────────────────────────────────────────────

type SQIDimensionsV2 struct {
	Mastery  float64 `json:"mastery"`
	Speed    float64 `json:"speed"`
	Risk     float64 `json:"risk"`
	Coverage float64 `json:"coverage"`
}

// DiagnosticPayloadV2 is the full object fed to the LLM.
// It contains scores, exam context, per-attempt profiles,
// per-concept breakdowns, and behavioral flags.
type DiagnosticPayloadV2 struct {
	// ── Engine version — set automatically by Analyze() ──────────
	Version string `json:"version"`

	// ── Scores (also shown to student / teacher) ──────────────────
	OverallSQI float64         `json:"overall_sqi"`
	Dimensions SQIDimensionsV2 `json:"dimensions"`

	// ── Exam-level summary ─────────────────────────────────────────
	ExamSummary ExamSummaryV2 `json:"exam_summary"`

	// ── How the student attempted the paper ────────────────────────
	AttemptProfile AttemptProfileV2 `json:"attempt_profile"`

	// ── Per-concept breakdown, sorted by priority ──────────────────
	ConceptProfiles []ConceptProfileV2 `json:"concept_profiles"`

	// ── Behavioral coaching signals ────────────────────────────────
	BehaviorFlags BehaviorFlagsV2 `json:"behavior_flags"`

	// ── Half-paper performance split ───────────────────────────────
	FirstHalfAccuracy  float64 `json:"first_half_accuracy"`
	SecondHalfAccuracy float64 `json:"second_half_accuracy"`
}

// ExamSummaryV2 is the high-level numbers — what teachers see at a glance.
type ExamSummaryV2 struct {
	ExamType           string  `json:"exam_type"`
	HasNegativeMarking bool    `json:"has_negative_marking"`
	TotalQuestions     int     `json:"total_questions"`
	Attempted          int     `json:"attempted"`
	Correct            int     `json:"correct"`
	Wrong              int     `json:"wrong"`
	Skipped            int     `json:"skipped"`
	Unseen             int     `json:"unseen"`
	TotalMarksEarned   float64 `json:"total_marks_earned"`
	TotalMarksLost     float64 `json:"total_marks_lost"`
	NetScore           float64 `json:"net_score"`
	MaxPossibleScore   float64 `json:"max_possible_score"`
	ScorePercent       float64 `json:"score_percent"`
}

// AttemptProfileV2 classifies every question attempt by type.
type AttemptProfileV2 struct {
	GuessedWrong   int `json:"guessed_wrong"`
	CarefullyWrong int `json:"carefully_wrong"`
	GuessedRight   int `json:"guessed_right"`
	CarefullyRight int `json:"carefully_right"`
	SeenAbandoned  int `json:"seen_abandoned"`
	NeverReached   int `json:"never_reached"`
	NegMarksFromGuess   float64 `json:"neg_marks_from_guess"`
	NegMarksFromCareful float64 `json:"neg_marks_from_careful"`
}

// ConceptProfileV2 is the per-topic diagnostic entry.
type ConceptProfileV2 struct {
	ConceptTag   string            `json:"concept_tag"`
	Subject      string            `json:"subject"`
	Status       ConceptStatusV2   `json:"status"`
	PriorityRank int               `json:"priority_rank"`
	Evidence     ConceptEvidenceV2 `json:"evidence"`
}

// ConceptStatusV2 is a human-readable classification computed by Go.
type ConceptStatusV2 string

const (
	StatusMasteredV2    ConceptStatusV2 = "mastered"
	StatusAlmostThereV2 ConceptStatusV2 = "almost_there"
	StatusConfusedV2    ConceptStatusV2 = "confused"
	StatusNotStudiedV2  ConceptStatusV2 = "not_studied"
	StatusNotReachedV2  ConceptStatusV2 = "not_reached"
)

// ConceptEvidenceV2 is the raw numbers behind a concept's status.
type ConceptEvidenceV2 struct {
	TotalQuestions   int     `json:"total_questions"`
	Attempted        int     `json:"attempted"`
	Correct          int     `json:"correct"`
	Wrong            int     `json:"wrong"`
	Skipped          int     `json:"skipped"`
	Unseen           int     `json:"unseen"`
	AccuracyPct      float64 `json:"accuracy_pct"`
	AvgTimeRatio     float64 `json:"avg_time_ratio"`
	NegMarksCost     float64 `json:"neg_marks_cost"`
	GuessCount       int     `json:"guess_count"`
	GenuineWrong     int     `json:"genuine_wrong"`
	ChangedToCorrect int     `json:"changed_to_correct"`
	ChangedToWrong   int     `json:"changed_to_wrong"`
	MasteryScore     float64 `json:"mastery_score"`
	PriorityScore    float64 `json:"priority_score"`
}

// BehaviorFlagsV2 are boolean coaching signals with a confidence weight.
type BehaviorFlagsV2 struct {
	PanicGuesser    BehaviorFlagV2 `json:"panic_guesser"`
	TimeMismanager  BehaviorFlagV2 `json:"time_mismanager"`
	Overconfident   BehaviorFlagV2 `json:"overconfident"`
	ReviewWasted    BehaviorFlagV2 `json:"review_wasted"`
	EarlyExhaustion BehaviorFlagV2 `json:"early_exhaustion"`
	RiskyAttempter  BehaviorFlagV2 `json:"risky_attempter"`
	StrongStarter   BehaviorFlagV2 `json:"strong_starter"`
}

// BehaviorFlagV2 pairs a detected behavior with a confidence level.
type BehaviorFlagV2 struct {
	Detected   bool    `json:"detected"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}
