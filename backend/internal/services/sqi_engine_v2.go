package services

import (
	"ai-student-diagnostic/backend/internal/helper"
	"math"
	"sort"
)

// ─────────────────────────────────────────────
// INPUT TYPES
// ─────────────────────────────────────────────

// QuestionMetaV2 describes a question in the exam paper.
// Order in the slice is assumed to be the order questions appeared in the paper.
type QuestionMetaV2 struct {
	QuestionID   int     `json:"question_id"`
	Marks        float64 `json:"marks"`
	NegMarks     float64 `json:"neg_marks"`
	Importance   string  `json:"importance"`
	Difficulty   string  `json:"difficulty"`
	Type         string  `json:"type"`
	ExpectedTime float64 `json:"expected_time"`
	ConceptTag   string  `json:"concept_tag"`
	Subject      string  `json:"subject"`
}

// AnswerLogV2 is what the student did on a question.
type AnswerLogV2 struct {
	QuestionID        int     `json:"question_id"`
	SelectedAnswer    string  `json:"selected_answer"`
	CorrectAnswer     string  `json:"correct_answer"`
	TimeSpent         float64 `json:"time_spent"`
	MarkedForReview   bool    `json:"marked_for_review"`
	Revisited         bool    `json:"revisited"`
	ChangedAnswer     bool    `json:"changed_answer"`
	WasInitiallyWrong bool    `json:"was_initially_wrong"`
	Seen              bool    `json:"seen"`
}

// ExamConfigV2 carries exam-level settings.
type ExamConfigV2 struct {
	ExamType           string  `json:"exam_type"`
	HasNegativeMarking bool    `json:"has_negative_marking"`
	TotalDuration      float64 `json:"total_duration"`
}

// ─────────────────────────────────────────────
// OUTPUT TYPES — SCORES
// ─────────────────────────────────────────────

// SQIDimensionsV2 holds the four individual dimension scores (0–100 each).
type SQIDimensionsV2 struct {
	Mastery  float64 `json:"mastery"`
	Speed    float64 `json:"speed"`
	Risk     float64 `json:"risk"`
	Coverage float64 `json:"coverage"`
}

// ─────────────────────────────────────────────
// OUTPUT TYPES — DIAGNOSTIC PAYLOAD
// ─────────────────────────────────────────────

// DiagnosticPayloadV2 is the full object fed to the LLM.
// It contains scores, exam context, per-attempt profiles,
// per-concept breakdowns, and behavioral flags.
type DiagnosticPayloadV2 struct {
	// ── Scores (also shown to student / teacher) ──────────────────
	OverallSQI float64          `json:"overall_sqi"`
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
	// Useful for detecting early exhaustion / panic in latter half.
	FirstHalfAccuracy  float64 `json:"first_half_accuracy"`  // % correct in first 50% of questions
	SecondHalfAccuracy float64 `json:"second_half_accuracy"` // % correct in second 50% of questions
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
// This is core LLM context: it shows *why* marks were lost.
type AttemptProfileV2 struct {
	// Wrong answers
	GuessedWrong   int `json:"guessed_wrong"`
	CarefullyWrong int `json:"carefully_wrong"`

	// Correct answers
	GuessedRight   int `json:"guessed_right"`
	CarefullyRight int `json:"carefully_right"`

	// Non-attempts
	SeenAbandoned int `json:"seen_abandoned"`
	NeverReached  int `json:"never_reached"`

	// Negative marks breakdown
	NegMarksFromGuess   float64 `json:"neg_marks_from_guess"`
	NegMarksFromCareful float64 `json:"neg_marks_from_careful"`
}

// ConceptProfileV2 is the per-topic diagnostic entry.
type ConceptProfileV2 struct {
	ConceptTag   string             `json:"concept_tag"`
	Subject      string             `json:"subject"`
	Status       ConceptStatusV2    `json:"status"`
	PriorityRank int                `json:"priority_rank"`
	Evidence     ConceptEvidenceV2 `json:"evidence"`
}

// ConceptStatusV2 is a human-readable classification computed by Go.
// The LLM uses this to decide what kind of plan to generate.
type ConceptStatusV2 string

const (
	StatusMasteredV2    ConceptStatusV2 = "mastered"     // knows it well and fast
	StatusAlmostThereV2 ConceptStatusV2 = "almost_there" // knows it but slow or inconsistent
	StatusConfusedV2    ConceptStatusV2 = "confused"     // attempted but mostly wrong — wrong mental model
	StatusNotStudiedV2  ConceptStatusV2 = "not_studied"  // mostly guessing or skipping — topic not covered
	StatusNotReachedV2  ConceptStatusV2 = "not_reached"  // majority unseen — time issue, not knowledge
)

// ConceptEvidenceV2 is the raw numbers behind a concept's status.
type ConceptEvidenceV2 struct {
	TotalQuestions int     `json:"total_questions"`
	Attempted      int     `json:"attempted"`
	Correct        int     `json:"correct"`
	Wrong          int     `json:"wrong"`
	Skipped        int     `json:"skipped"`
	Unseen         int     `json:"unseen"`
	AccuracyPct    float64 `json:"accuracy_pct"`
	AvgTimeRatio   float64 `json:"avg_time_ratio"`
	NegMarksCost   float64 `json:"neg_marks_cost"`
	GuessCount     int     `json:"guess_count"`
	GenuineWrong   int     `json:"genuine_wrong"`
	ChangedToCorrect int   `json:"changed_to_correct"`
	ChangedToWrong   int     `json:"changed_to_wrong"`
	MasteryScore    float64 `json:"mastery_score"`
	PriorityScore   float64 `json:"priority_score"`
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

// ─────────────────────────────────────────────
// INTERNAL TYPES
// ─────────────────────────────────────────────

type outcomeType string

const (
	outcomeCorrect outcomeType = "correct"
	outcomeWrong   outcomeType = "wrong"
	outcomeSkipped outcomeType = "skipped" // seen, no answer
	outcomeUnseen  outcomeType = "unseen"
)

type attemptType string

const (
	attemptGuess   attemptType = "guess"   // timeRatio < 0.4 AND wrong
	attemptGenuine attemptType = "genuine" // anything else when answered
	attemptNone    attemptType = "none"    // skipped or unseen
)

type timeBucket string

const (
	timeFast     timeBucket = "fast"      // < 0.4× expected
	timeNormal   timeBucket = "normal"    // 0.4–1.5× expected
	timeSlow     timeBucket = "slow"      // 1.5–2.5× expected
	timeVerySlow timeBucket = "very_slow" // > 2.5× expected
)

// questionResult is the per-question computed result used internally.
type questionResult struct {
	QuestionID   int
	Outcome      outcomeType
	AttemptKind  attemptType
	TimeBucket   timeBucket
	TimeRatio    float64
	WeightFactor float64
	MarksEarned  float64
	MarksLost    float64 // from negative marking, always ≥ 0
	IsCorrect    bool
	Importance   string
	Difficulty   string
	ConceptTag   string
	Subject      string
	// behavioral extras
	MarkedForReview   bool
	Revisited         bool
	ChangedAnswer     bool
	WasInitiallyWrong bool
}

// conceptAggregateV2 collects all question results under a concept tag.
type conceptAggregateV2 struct {
	Subject   string
	Results   []questionResult
	Questions []QuestionMetaV2
}

// ─────────────────────────────────────────────
// MAIN ENTRY POINT
// ─────────────────────────────────────────────

// Analyze is the single public function. It takes the question list,
// answer log, and exam config, and returns the full DiagnosticPayloadV2.
func Analyze(questions []QuestionMetaV2, answers []AnswerLogV2, cfg ExamConfigV2) DiagnosticPayloadV2 {
	answerMap := helper.BuildAnswerMapV2(answers, func(a AnswerLogV2) int {
		return a.QuestionID
	})

	// ── Step 1: Compute per-question results ───────────────────────
	results := make([]questionResult, 0, len(questions))
	for _, q := range questions {
		ans, found := answerMap[q.QuestionID]
		r := computeQuestionResult(q, ans, found, cfg)
		results = append(results, r)
	}

	// ── Step 2: Exam summary ────────────────────────────────────────
	summary := buildExamSummary(results, questions, cfg)

	// ── Step 3: Attempt profile ─────────────────────────────────────
	profile := buildAttemptProfile(results)

	// ── Step 4: SQI dimensions ──────────────────────────────────────
	dims := computeDimensions(results, summary, cfg)
	overallSQI := helper.Round2V2(0.35*dims.Mastery + 0.25*dims.Speed + 0.25*dims.Risk + 0.15*dims.Coverage)

	// ── Step 5: Concept profiles ────────────────────────────────────
	conceptMap := groupByConcept(results, questions)
	conceptProfiles := buildConceptProfiles(conceptMap)

	// ── Step 6: Behavioral flags ────────────────────────────────────
	flags := detectBehaviorFlags(results, profile, questions)

	// ── Step 7: Half-paper accuracy ─────────────────────────────────
	firstAcc, secondAcc := computeHalfAccuracy(results)

	return DiagnosticPayloadV2{
		OverallSQI:         overallSQI,
		Dimensions:         dims,
		ExamSummary:        summary,
		AttemptProfile:     profile,
		ConceptProfiles:    conceptProfiles,
		BehaviorFlags:      flags,
		FirstHalfAccuracy:  firstAcc,
		SecondHalfAccuracy: secondAcc,
	}
}

// ─────────────────────────────────────────────
// STEP 1 — PER-QUESTION RESULT
// ─────────────────────────────────────────────

func computeQuestionResult(q QuestionMetaV2, ans AnswerLogV2, found bool, cfg ExamConfigV2) questionResult {
	r := questionResult{
		QuestionID:   q.QuestionID,
		WeightFactor: helper.GetSQIV2ImportanceWeight(q.Importance) * helper.GetSQIV2DifficultyWeight(q.Difficulty) * helper.GetSQIV2TypeWeight(q.Type),
		Importance:   q.Importance,
		Difficulty:   q.Difficulty,
		ConceptTag:   helper.CoalesceV2(q.ConceptTag, "uncategorized"),
		Subject:      q.Subject,
	}

	// Not found in answer log or never seen → unseen
	if !found || !ans.Seen {
		r.Outcome = outcomeUnseen
		r.AttemptKind = attemptNone
		r.TimeBucket = timeFast // irrelevant but needs a value
		return r
	}

	r.MarkedForReview = ans.MarkedForReview
	r.Revisited = ans.Revisited
	r.ChangedAnswer = ans.ChangedAnswer
	r.WasInitiallyWrong = ans.WasInitiallyWrong

	// Seen but no answer submitted → skipped
	if ans.SelectedAnswer == "" {
		r.Outcome = outcomeSkipped
		r.AttemptKind = attemptNone
		r.TimeBucket = timeNormal
		return r
	}

	// Answered
	r.IsCorrect = ans.SelectedAnswer == ans.CorrectAnswer
	r.TimeRatio = helper.SafeDivideV2(ans.TimeSpent, q.ExpectedTime)
	r.TimeBucket = classifyTime(r.TimeRatio)

	if r.IsCorrect {
		r.Outcome = outcomeCorrect
		r.MarksEarned = q.Marks
		// Fast+correct = fluent, slow+correct = grinding — both are genuine
		r.AttemptKind = attemptGenuine
	} else {
		r.Outcome = outcomeWrong
		if cfg.HasNegativeMarking {
			r.MarksLost = q.NegMarks
		}
		// Guess = answered very fast AND wrong
		if r.TimeBucket == timeFast {
			r.AttemptKind = attemptGuess
		} else {
			r.AttemptKind = attemptGenuine
		}
	}

	return r
}

func classifyTime(ratio float64) timeBucket {
	switch {
	case ratio < 0.4:
		return timeFast
	case ratio <= 1.5:
		return timeNormal
	case ratio <= 2.5:
		return timeSlow
	default:
		return timeVerySlow
	}
}

// ─────────────────────────────────────────────
// STEP 2 — EXAM SUMMARY
// ─────────────────────────────────────────────

func buildExamSummary(results []questionResult, questions []QuestionMetaV2, cfg ExamConfigV2) ExamSummaryV2 {
	s := ExamSummaryV2{
		ExamType:           cfg.ExamType,
		HasNegativeMarking: cfg.HasNegativeMarking,
		TotalQuestions:     len(results),
	}

	var maxPossible float64
	for _, q := range questions {
		maxPossible += q.Marks
	}
	s.MaxPossibleScore = maxPossible

	for _, r := range results {
		switch r.Outcome {
		case outcomeCorrect:
			s.Correct++
			s.Attempted++
			s.TotalMarksEarned += r.MarksEarned
		case outcomeWrong:
			s.Wrong++
			s.Attempted++
			s.TotalMarksLost += r.MarksLost
		case outcomeSkipped:
			s.Skipped++
		case outcomeUnseen:
			s.Unseen++
		}
	}

	s.NetScore = helper.Round2V2(s.TotalMarksEarned - s.TotalMarksLost)
	if s.MaxPossibleScore > 0 {
		s.ScorePercent = helper.Round2V2(s.NetScore / s.MaxPossibleScore * 100)
	}
	s.TotalMarksEarned = helper.Round2V2(s.TotalMarksEarned)
	s.TotalMarksLost = helper.Round2V2(s.TotalMarksLost)
	return s
}

// ─────────────────────────────────────────────
// STEP 3 — ATTEMPT PROFILE
// ─────────────────────────────────────────────

func buildAttemptProfile(results []questionResult) AttemptProfileV2 {
	var p AttemptProfileV2
	for _, r := range results {
		switch r.Outcome {
		case outcomeCorrect:
			if r.TimeBucket == timeFast {
				p.GuessedRight++
			} else {
				p.CarefullyRight++
			}
		case outcomeWrong:
			if r.AttemptKind == attemptGuess {
				p.GuessedWrong++
				p.NegMarksFromGuess += r.MarksLost
			} else {
				p.CarefullyWrong++
				p.NegMarksFromCareful += r.MarksLost
			}
		case outcomeSkipped:
			p.SeenAbandoned++
		case outcomeUnseen:
			p.NeverReached++
		}
	}
	p.NegMarksFromGuess = helper.Round2V2(p.NegMarksFromGuess)
	p.NegMarksFromCareful = helper.Round2V2(p.NegMarksFromCareful)
	return p
}

// ─────────────────────────────────────────────
// STEP 4 — SQI DIMENSIONS
// ─────────────────────────────────────────────

func computeDimensions(results []questionResult, summary ExamSummaryV2, cfg ExamConfigV2) SQIDimensionsV2 {
	return SQIDimensionsV2{
		Mastery:  helper.Round2V2(computeMastery(results)),
		Speed:    helper.Round2V2(computeSpeed(results)),
		Risk:     helper.Round2V2(computeRisk(results, summary, cfg)),
		Coverage: helper.Round2V2(computeCoverage(results)),
	}
}

// Mastery — weighted accuracy.
// Correct on hard/important questions scores more.
// Wrong on easy questions penalized more.
// Skipped = 30% penalty, unseen = no penalty (time issue, not knowledge).
func computeMastery(results []questionResult) float64 {
	var weightedScore, weightedMax float64

	for _, r := range results {
		w := r.WeightFactor
		if w == 0 {
			w = 1
		}

		switch r.Outcome {
		case outcomeCorrect:
			weightedScore += 1.0 * w
			weightedMax += 1.0 * w
		case outcomeWrong:
			// Wrong on easy = bigger penalty
			penalty := 0.0
			switch r.Difficulty {
			case "E":
				penalty = -0.4
			case "M":
				penalty = -0.2
			case "H":
				penalty = -0.1
			}
			weightedScore += penalty * w
			weightedMax += 1.0 * w
		case outcomeSkipped:
			// Skipped = mild mastery hit (made a choice, didn't attempt)
			weightedScore += -0.15 * w
			weightedMax += 1.0 * w
		case outcomeUnseen:
			// Unseen does not count against mastery — it's a coverage issue
			weightedMax += 1.0 * w
		}
	}

	if weightedMax == 0 {
		return 0
	}
	return helper.ClampV2((weightedScore/weightedMax)*100, 0, 100)
}

// Speed — quality of time usage on attempted questions.
// Matrix: outcome × timeBucket → score (0–100)
// Fast+correct = 100 (fluent), slow+wrong = 15 (wasted time AND marks)
func computeSpeed(results []questionResult) float64 {
	speedMatrix := map[outcomeType]map[timeBucket]float64{
		outcomeCorrect: {
			timeFast:     100,
			timeNormal:   80,
			timeSlow:     55,
			timeVerySlow: 30,
		},
		outcomeWrong: {
			timeFast:     25, // guessing — at least didn't waste much time
			timeNormal:   40,
			timeSlow:     20,
			timeVerySlow: 10,
		},
	}

	var totalScore, totalWeight float64
	for _, r := range results {
		if r.Outcome != outcomeCorrect && r.Outcome != outcomeWrong {
			continue // skipped/unseen don't affect speed score
		}
		w := helper.GetSQIV2ImportanceWeight(r.Importance)
		score := speedMatrix[r.Outcome][r.TimeBucket]
		totalScore += score * w
		totalWeight += w
	}

	if totalWeight == 0 {
		return 50 // neutral default if no attempted questions
	}
	return helper.ClampV2(totalScore/totalWeight, 0, 100)
}

// Risk — how well the student managed negative marking.
// Starts at 100, deductions for guessing and bad behavioral patterns,
// small bonuses for good risk awareness.
func computeRisk(results []questionResult, summary ExamSummaryV2, cfg ExamConfigV2) float64 {
	score := 100.0

	if !cfg.HasNegativeMarking {
		// Without negative marking, risk is purely about coverage of easy questions
		// Penalize only for leaving easy questions unseen
		for _, r := range results {
			if r.Outcome == outcomeUnseen && r.Difficulty == "E" {
				score -= 3 * helper.GetSQIV2ImportanceWeight(r.Importance)
			}
		}
		return helper.ClampV2(score, 0, 100)
	}

	for _, r := range results {
		iw := helper.GetSQIV2ImportanceWeight(r.Importance)

		switch {
		case r.AttemptKind == attemptGuess:
			// Fast + wrong = gambling. Heavier deduction for important questions.
			score -= 6 * iw

		case r.Outcome == outcomeWrong && r.AttemptKind == attemptGenuine:
			// Genuinely wrong — smaller risk deduction (acceptable risk taking)
			score -= 2 * iw

		case r.ChangedAnswer && r.WasInitiallyWrong && r.IsCorrect:
			// Changed wrong → correct = good self-awareness, small bonus
			score += 2

		case r.ChangedAnswer && !r.WasInitiallyWrong && !r.IsCorrect:
			// Changed correct → wrong = overconfidence, deduct
			score -= 5

		case r.Outcome == outcomeUnseen && r.Difficulty == "E":
			// Left easy questions unseen = poor time/risk management
			score -= 3 * iw

		case r.Outcome == outcomeSkipped && r.Difficulty == "H":
			// Skipped hard questions when negative marking is on = good risk management
			score += 1
		}
	}

	return helper.ClampV2(score, 0, 100)
}

// Coverage — how much of the paper the student engaged with.
// Unseen high-importance questions penalized heavily.
// Seen-but-skipped = 50% credit (conscious choice is better than not reaching).
func computeCoverage(results []questionResult) float64 {
	var weightedEngaged, weightedTotal float64

	for _, r := range results {
		// Weight by importance only — difficulty doesn't affect coverage expectation
		w := helper.GetSQIV2ImportanceWeight(r.Importance)
		weightedTotal += w

		switch r.Outcome {
		case outcomeCorrect, outcomeWrong:
			weightedEngaged += 1.0 * w
		case outcomeSkipped:
			weightedEngaged += 0.5 * w // saw it, made a choice
		case outcomeUnseen:
			weightedEngaged += 0.0 // missed entirely
		}
	}

	if weightedTotal == 0 {
		return 0
	}
	return helper.ClampV2((weightedEngaged/weightedTotal)*100, 0, 100)
}

// ─────────────────────────────────────────────
// STEP 5 — CONCEPT PROFILES
// ─────────────────────────────────────────────

func groupByConcept(results []questionResult, questions []QuestionMetaV2) map[string]*conceptAggregateV2 {
	subjectMap := make(map[int]string)
	for _, q := range questions {
		subjectMap[q.QuestionID] = q.Subject
	}

	concepts := make(map[string]*conceptAggregateV2)
	for _, r := range results {
		tag := r.ConceptTag
		if _, ok := concepts[tag]; !ok {
			concepts[tag] = &conceptAggregateV2{Subject: r.Subject}
		}
		concepts[tag].Results = append(concepts[tag].Results, r)
	}
	return concepts
}

func buildConceptProfiles(concepts map[string]*conceptAggregateV2) []ConceptProfileV2 {
	profiles := make([]ConceptProfileV2, 0, len(concepts))

	for tag, agg := range concepts {
		ev := computeConceptEvidence(agg.Results)
		status := classifyConceptStatus(ev)
		priority := computeConceptPriority(ev, status)
		ev.MasteryScore = helper.Round2V2(computeConceptMastery(agg.Results))
		ev.PriorityScore = helper.Round2V2(priority)

		profiles = append(profiles, ConceptProfileV2{
			ConceptTag: tag,
			Subject:    agg.Subject,
			Status:     status,
			Evidence:   ev,
		})
	}

	// Sort by priority descending (highest priority = needs most work = rank 1)
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Evidence.PriorityScore > profiles[j].Evidence.PriorityScore
	})
	for i := range profiles {
		profiles[i].PriorityRank = i + 1
	}

	return profiles
}

func computeConceptEvidence(results []questionResult) ConceptEvidenceV2 {
	ev := ConceptEvidenceV2{TotalQuestions: len(results)}

	var timeRatioSum float64
	var timeRatioCount int

	for _, r := range results {
		switch r.Outcome {
		case outcomeCorrect:
			ev.Attempted++
			ev.Correct++
			timeRatioSum += r.TimeRatio
			timeRatioCount++
		case outcomeWrong:
			ev.Attempted++
			ev.Wrong++
			ev.NegMarksCost += r.MarksLost
			timeRatioSum += r.TimeRatio
			timeRatioCount++
			if r.AttemptKind == attemptGuess {
				ev.GuessCount++
			} else {
				ev.GenuineWrong++
			}
		case outcomeSkipped:
			ev.Skipped++
		case outcomeUnseen:
			ev.Unseen++
		}

		if r.ChangedAnswer {
			if r.WasInitiallyWrong && r.IsCorrect {
				ev.ChangedToCorrect++
			} else if !r.WasInitiallyWrong && !r.IsCorrect {
				ev.ChangedToWrong++
			}
		}
	}

	if ev.Attempted > 0 {
		ev.AccuracyPct = helper.Round2V2(float64(ev.Correct) / float64(ev.Attempted) * 100)
	}
	if timeRatioCount > 0 {
		ev.AvgTimeRatio = helper.Round2V2(timeRatioSum / float64(timeRatioCount))
	}
	ev.NegMarksCost = helper.Round2V2(ev.NegMarksCost)
	return ev
}

// classifyConceptStatus applies threshold rules to produce a ConceptStatusV2.
// The LLM uses this label to decide what kind of remediation to suggest.
func classifyConceptStatus(ev ConceptEvidenceV2) ConceptStatusV2 {
	total := ev.TotalQuestions
	if total == 0 {
		return StatusNotStudiedV2
	}

	unseenRatio := float64(ev.Unseen) / float64(total)
	attemptedRatio := float64(ev.Attempted) / float64(total)

	// Most questions never reached → time problem, not knowledge problem
	if unseenRatio >= 0.6 {
		return StatusNotReachedV2
	}

	// Very few attempts → student hasn't studied this
	if attemptedRatio < 0.3 && ev.GuessCount > ev.GenuineWrong {
		return StatusNotStudiedV2
	}

	// Attempted enough questions to judge knowledge
	if ev.Attempted >= 2 {
		switch {
		case ev.AccuracyPct >= 80 && ev.AvgTimeRatio <= 1.5:
			return StatusMasteredV2

		case ev.AccuracyPct >= 80 && ev.AvgTimeRatio > 1.5:
			// Knows it but too slow — almost there
			return StatusAlmostThereV2

		case ev.AccuracyPct >= 50:
			return StatusAlmostThereV2

		case ev.AccuracyPct < 50 && ev.GenuineWrong > ev.GuessCount:
			// Tried genuinely but mostly wrong = wrong mental model
			return StatusConfusedV2

		default:
			return StatusNotStudiedV2
		}
	}

	// Too few attempts to classify confidently
	if ev.AccuracyPct >= 80 {
		return StatusAlmostThereV2 // don't claim mastered on 1–2 questions
	}
	return StatusConfusedV2
}

// computeConceptPriority returns a 0–1 composite score.
// Higher = needs more attention from the LLM and teacher.
func computeConceptPriority(ev ConceptEvidenceV2, status ConceptStatusV2) float64 {
	// Status contributes most weight
	statusScore := map[ConceptStatusV2]float64{
		StatusConfusedV2:    1.0,
		StatusNotStudiedV2:  0.9,
		StatusNotReachedV2:  0.7,
		StatusAlmostThereV2: 0.5,
		StatusMasteredV2:    0.1,
	}[status]

	// Negative marks cost adds urgency
	negMarksFactor := math.Min(ev.NegMarksCost/10.0, 1.0) // cap contribution at 10 marks lost

	// Accuracy inverse — lower accuracy = higher priority
	accuracyFactor := 1 - (ev.AccuracyPct / 100)

	// Genuine wrong (knowledge gap) adds more priority than guesses
	genuineWrongFactor := 0.0
	if ev.Attempted > 0 {
		genuineWrongFactor = math.Min(float64(ev.GenuineWrong)/float64(ev.Attempted), 1.0)
	}

	priority := 0.40*statusScore +
		0.25*accuracyFactor +
		0.20*genuineWrongFactor +
		0.15*negMarksFactor

	return helper.ClampV2(priority, 0, 1)
}

func computeConceptMastery(results []questionResult) float64 {
	// Reuse the global mastery logic but scoped to this concept's results
	return computeMastery(results)
}

// ─────────────────────────────────────────────
// STEP 6 — BEHAVIORAL FLAGS
// ─────────────────────────────────────────────

func detectBehaviorFlags(results []questionResult, profile AttemptProfileV2, questions []QuestionMetaV2) BehaviorFlagsV2 {
	var flags BehaviorFlagsV2
	total := len(results)
	if total == 0 {
		return flags
	}

	// ── Panic guesser ──────────────────────────────────────────────
	// More than 30% of wrong answers were guesses
	if profile.CarefullyWrong+profile.GuessedWrong > 0 {
		guessRatio := float64(profile.GuessedWrong) / float64(profile.CarefullyWrong+profile.GuessedWrong)
		if guessRatio > 0.3 {
			flags.PanicGuesser = BehaviorFlagV2{
				Detected:   true,
				Confidence: helper.Round2V2(guessRatio),
				Evidence:   helper.FormatfV2("%d of %d wrong answers were guesses (fast+wrong)", profile.GuessedWrong, profile.GuessedWrong+profile.CarefullyWrong),
			}
		}
	}

	// ── Time mismanager ─────────────────────────────────────────────
	// Missed important easy questions (unseen, difficulty E, importance high/medium)
	var missedImportantEasy int
	for _, r := range results {
		if r.Outcome == outcomeUnseen && r.Difficulty == "E" && (r.Importance == "high" || r.Importance == "medium") {
			missedImportantEasy++
		}
	}
	if missedImportantEasy >= 3 {
		conf := math.Min(float64(missedImportantEasy)/10.0, 1.0)
		flags.TimeMismanager = BehaviorFlagV2{
			Detected:   true,
			Confidence: helper.Round2V2(conf),
			Evidence:   helper.FormatfV2("%d important easy questions never reached due to time", missedImportantEasy),
		}
	}

	// ── Overconfident ───────────────────────────────────────────────
	// Changed correct answers to wrong more than once
	var correctToWrong int
	for _, r := range results {
		if r.ChangedAnswer && !r.WasInitiallyWrong && !r.IsCorrect {
			correctToWrong++
		}
	}
	if correctToWrong >= 2 {
		flags.Overconfident = BehaviorFlagV2{
			Detected:   true,
			Confidence: helper.Round2V2(math.Min(float64(correctToWrong)/5.0, 1.0)),
			Evidence:   helper.FormatfV2("changed correct answers to wrong %d times", correctToWrong),
		}
	}

	// ── Review wasted ───────────────────────────────────────────────
	// Marked many questions for review but didn't revisit most of them
	var markedCount, revisitedAfterMark int
	for _, r := range results {
		if r.MarkedForReview {
			markedCount++
			if r.Revisited {
				revisitedAfterMark++
			}
		}
	}
	if markedCount >= 3 {
		revisitRatio := float64(revisitedAfterMark) / float64(markedCount)
		if revisitRatio < 0.5 {
			flags.ReviewWasted = BehaviorFlagV2{
				Detected:   true,
				Confidence: helper.Round2V2(1 - revisitRatio),
				Evidence:   helper.FormatfV2("marked %d for review but only revisited %d (%.0f%%)", markedCount, revisitedAfterMark, revisitRatio*100),
			}
		}
	}

	// ── Risky attempter ─────────────────────────────────────────────
	// Attempted hard questions wrong AND left easy questions unseen
	var hardWrong, easyUnseen int
	for _, r := range results {
		if r.Outcome == outcomeWrong && r.Difficulty == "H" {
			hardWrong++
		}
		if r.Outcome == outcomeUnseen && r.Difficulty == "E" {
			easyUnseen++
		}
	}
	if hardWrong >= 2 && easyUnseen >= 2 {
		conf := math.Min(float64(hardWrong+easyUnseen)/10.0, 1.0)
		flags.RiskyAttempter = BehaviorFlagV2{
			Detected:   true,
			Confidence: helper.Round2V2(conf),
			Evidence:   helper.FormatfV2("attempted %d hard questions (wrong) while %d easy questions were left unseen", hardWrong, easyUnseen),
		}
	}

	// ── Early exhaustion & strong starter ──────────────────────────
	firstAcc, secondAcc := computeHalfAccuracy(results)
	drop := firstAcc - secondAcc

	if drop >= 25 && secondAcc < 50 {
		flags.EarlyExhaustion = BehaviorFlagV2{
			Detected:   true,
			Confidence: helper.Round2V2(math.Min(drop/50.0, 1.0)),
			Evidence:   helper.FormatfV2("accuracy dropped from %.0f%% (first half) to %.0f%% (second half)", firstAcc, secondAcc),
		}
	}

	if drop >= 20 {
		flags.StrongStarter = BehaviorFlagV2{
			Detected:   true,
			Confidence: helper.Round2V2(math.Min(drop/40.0, 1.0)),
			Evidence:   helper.FormatfV2("first half accuracy %.0f%% vs second half %.0f%%", firstAcc, secondAcc),
		}
	}

	return flags
}

// ─────────────────────────────────────────────
// STEP 7 — HALF-PAPER ACCURACY
// ─────────────────────────────────────────────

func computeHalfAccuracy(results []questionResult) (float64, float64) {
	if len(results) == 0 {
		return 0, 0
	}
	mid := len(results) / 2
	first := results[:mid]
	second := results[mid:]

	return halfAcc(first), halfAcc(second)
}

func halfAcc(results []questionResult) float64 {
	var correct, attempted int
	for _, r := range results {
		if r.Outcome == outcomeCorrect || r.Outcome == outcomeWrong {
			attempted++
			if r.Outcome == outcomeCorrect {
				correct++
			}
		}
	}
	if attempted == 0 {
		return 0
	}
	return helper.Round2V2(float64(correct) / float64(attempted) * 100)
}

// ─────────────────────────────────────────────
// WEIGHT HELPERS
// ─────────────────────────────────────────────

// ─────────────────────────────────────────────
// UTILITY
// ─────────────────────────────────────────────

// ─────────────────────────────────────────────
// COMPATIBILITY LAYER
// ─────────────────────────────────────────────

// AnalyzeFromLegacy allows calling the V2 engine using legacy V1 types.
// This is provided to help the existing app transition to the new engine.
func AnalyzeFromLegacy(questions []QuestionMeta, answers []AnswerLog, cfg ExamConfigV2) DiagnosticPayloadV2 {
	v2Qs := make([]QuestionMetaV2, len(questions))
	for i, q := range questions {
		v2Qs[i] = QuestionMetaV2{
			QuestionID:   q.QuestionID,
			Marks:        q.Marks,
			NegMarks:     q.NegMarks,
			Importance:   q.Importance,
			Difficulty:   q.Difficulty,
			Type:         q.Type,
			ExpectedTime: q.ExpectedTime,
			ConceptTag:   q.ConceptTag,
			Subject:      "Uncategorized",
		}
	}

	v2Ans := make([]AnswerLogV2, len(answers))
	for i, a := range answers {
		v2Ans[i] = AnswerLogV2{
			QuestionID:        a.QuestionID,
			SelectedAnswer:    a.SelectedAnswer,
			CorrectAnswer:     a.CorrectAnswer,
			TimeSpent:         a.TimeSpent,
			MarkedForReview:   a.MarkedForReview,
			Revisited:         a.Revisited,
			ChangedAnswer:     a.ChangedAnswer,
			WasInitiallyWrong: a.WasInitiallyWrong,
			Seen:              a.Seen,
		}
	}

	return Analyze(v2Qs, v2Ans, cfg)
}
