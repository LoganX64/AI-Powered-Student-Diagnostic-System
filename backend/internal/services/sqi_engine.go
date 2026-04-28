package services

import "math"

type QuestionMeta struct {
	QuestionID   int
	Marks        float64
	NegMarks     float64
	Importance   string
	Difficulty   string
	Type         string
	ExpectedTime float64
}

type AnswerLog struct {
	QuestionID        int
	SelectedAnswer    string
	CorrectAnswer     string
	TimeSpent         float64
	MarkedForReview   bool
	Revisited         bool
	ChangedAnswer     bool
	WasInitiallyWrong bool
	Seen              bool // NEW
}

type SQIResult struct {
	OverallSQI float64
}

func CalculateSQI(questions []QuestionMeta, answers []AnswerLog) SQIResult {

	// map answers
	answerMap := make(map[int]AnswerLog)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	var totalWeighted float64
	var maxPossible float64
	var minPossible float64

	for _, q := range questions {

		ans, attempted := answerMap[q.QuestionID]

		importanceW := getImportanceWeight(q.Importance)
		difficultyW := getDifficultyWeight(q.Difficulty)
		typeW := getTypeWeight(q.Type)

		weightFactor := importanceW * difficultyW * typeW

		var weighted float64

		// case: NOT SEEN (unattempted and unseen)
		if !attempted {

			// treat as NOT SEEN (worst case)
			base := -0.5 * q.NegMarks // configurable

			weighted = base * weightFactor

			totalWeighted += weighted
			minPossible += weighted
			maxPossible += q.Marks * weightFactor
			continue
		}

		// case: ANSWERED (attempted)

		isCorrect := ans.SelectedAnswer == ans.CorrectAnswer

		var base float64
		if isCorrect {
			base = q.Marks
		} else {
			base = -q.NegMarks
		}

		weighted = base * weightFactor

		// time adjustment (configurable)
		timeRatio := safeDivide(ans.TimeSpent, q.ExpectedTime)
		if timeRatio > 1 {
			weighted *= 1 / timeRatio
		}

		// behavioral adjustments (configurable)

		// doubtful but wrong
		if ans.MarkedForReview && !isCorrect {
			weighted *= 0.9
		}

		// corrected mistake (scaled properly)
		if ans.Revisited && ans.ChangedAnswer && isCorrect && ans.WasInitiallyWrong {
			bonus := 0.2 * q.Marks * weightFactor
			weighted += bonus
		}

		totalWeighted += weighted

		// track bounds
		maxPossible += q.Marks * weightFactor
		minPossible += (-q.NegMarks * weightFactor)
	}

	// normalize to 0-100
	rawPCT := 0.0
	rangeVal := maxPossible - minPossible

	if rangeVal > 0 {
		rawPCT = (totalWeighted - minPossible) / rangeVal * 100
	}

	rawPCT = clamp(rawPCT, 0, 100)

	return SQIResult{
		OverallSQI: Round(rawPCT, 2),
	}
}

type SQIAnalysis struct {
	OverallSQI float64

	Accuracy   AccuracyMetrics
	Time       TimeMetrics
	Difficulty DifficultyMetrics
	Behavior   BehaviorMetrics
	Skipping   SkippingMetrics
	Efficiency EfficiencyMetrics
}

type AccuracyMetrics struct {
	TotalQuestions int
	Attempted      int
	Correct        int
	Wrong          int
	Skipped        int
	AccuracyPct    float64
}

type TimeMetrics struct {
	TotalTime          float64
	TotalExpectedTime  float64
	AvgTimePerQuestion float64
	ExpectedAvgTime    float64

	SlowCount     int // >1.5x
	VerySlowCount int // >2x
	FastCount     int // <0.5x
}

type DifficultyStat struct {
	Total   int
	Correct int
	Wrong   int
	TimeSum float64
}

type DifficultyMetrics struct {
	Easy   DifficultyStat
	Medium DifficultyStat
	Hard   DifficultyStat
}

type BehaviorMetrics struct {
	MarkedForReview int
	AnswerChanged   int

	WrongToCorrect int
	CorrectToWrong int
}

type SkippingMetrics struct {
	NotSeen         int
	SeenNotAnswered int
	Answered        int
}

type EfficiencyMetrics struct {
	FastAndCorrect int
	FastAndWrong   int
	SlowAndCorrect int
	SlowAndWrong   int
}

func CalculateSQIAnalysis(questions []QuestionMeta, answers []AnswerLog) SQIAnalysis {

	answerMap := make(map[int]AnswerLog)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	var totalWeighted, maxPossible, minPossible float64

	var acc AccuracyMetrics
	var timeM TimeMetrics
	var beh BehaviorMetrics
	var skip SkippingMetrics
	var eff EfficiencyMetrics
	var diff DifficultyMetrics

	acc.TotalQuestions = len(questions)

	for _, q := range questions {

		ans, attempted := answerMap[q.QuestionID]

		importanceW := getImportanceWeight(q.Importance)
		difficultyW := getDifficultyWeight(q.Difficulty)
		typeW := getTypeWeight(q.Type)

		weightFactor := importanceW * difficultyW * typeW

		var weighted float64
		var isCorrect bool
		var timeRatio float64

		// not attempted
		if !attempted {
			skip.NotSeen++

			base := -0.5 * q.NegMarks
			weighted = base * weightFactor

			totalWeighted += weighted
			minPossible += weighted
			maxPossible += q.Marks * weightFactor
			continue
		}

		// seen skipped not answered

		if ans.Seen && ans.SelectedAnswer == "" {
			// skipped
			skip.SeenNotAnswered++
			acc.Skipped++

			base := -0.75 * q.NegMarks
			weighted = base * weightFactor

			totalWeighted += weighted
			minPossible += weighted
			maxPossible += q.Marks * weightFactor
			continue
		}

		// answered
		skip.Answered++
		acc.Attempted++

		isCorrect = ans.SelectedAnswer == ans.CorrectAnswer

		var base float64
		if isCorrect {
			base = q.Marks
			acc.Correct++
		} else {
			base = -q.NegMarks
			acc.Wrong++
		}

		weighted = base * weightFactor

		// time
		timeRatio = safeDivide(ans.TimeSpent, q.ExpectedTime)

		timeM.TotalTime += ans.TimeSpent
		timeM.TotalExpectedTime += q.ExpectedTime

		if timeRatio > 2 {
			timeM.VerySlowCount++
		} else if timeRatio > 1.5 {
			timeM.SlowCount++
		} else if timeRatio < 0.5 {
			timeM.FastCount++
		}

		if timeRatio > 1 {
			weighted *= 1 / timeRatio
		}

		// behavioral adjustments and tracking
		if ans.MarkedForReview {
			beh.MarkedForReview++
		}
		if ans.ChangedAnswer {
			beh.AnswerChanged++
		}
		if ans.WasInitiallyWrong && isCorrect {
			beh.WrongToCorrect++
		}
		if !ans.WasInitiallyWrong && !isCorrect {
			beh.CorrectToWrong++
		}

		if ans.MarkedForReview && !isCorrect {
			weighted *= 0.9
		}

		if ans.Revisited && ans.ChangedAnswer && isCorrect && ans.WasInitiallyWrong {
			bonus := 0.2 * q.Marks * weightFactor
			weighted += bonus
		}

		// efficiency breakdown
		if timeRatio < 0.5 && isCorrect {
			eff.FastAndCorrect++
		}
		if timeRatio < 0.5 && !isCorrect {
			eff.FastAndWrong++
		}
		if timeRatio > 1.5 && isCorrect {
			eff.SlowAndCorrect++
		}
		if timeRatio > 1.5 && !isCorrect {
			eff.SlowAndWrong++
		}
		// difficulty breakdown
		var d *DifficultyStat
		switch q.Difficulty {
		case "E":
			d = &diff.Easy
		case "M":
			d = &diff.Medium
		case "H":
			d = &diff.Hard
		default:
			d = &diff.Medium
		}

		d.Total++
		d.TimeSum += ans.TimeSpent
		if isCorrect {
			d.Correct++
		} else {
			d.Wrong++
		}

		// final weighted score
		totalWeighted += weighted
		maxPossible += q.Marks * weightFactor
		minPossible += (-q.NegMarks * weightFactor)
	}

	// final calculations

	rangeVal := maxPossible - minPossible
	rawPCT := 0.0
	if rangeVal > 0 {
		rawPCT = (totalWeighted - minPossible) / rangeVal * 100
	}
	rawPCT = clamp(rawPCT, 0, 100)

	// accuracy %
	if acc.Attempted > 0 {
		acc.AccuracyPct = (float64(acc.Correct) / float64(acc.Attempted)) * 100
	}

	// time averages
	if acc.TotalQuestions > 0 {
		timeM.AvgTimePerQuestion = timeM.TotalTime / float64(acc.TotalQuestions)
		timeM.ExpectedAvgTime = timeM.TotalExpectedTime / float64(acc.TotalQuestions)
	}

	return SQIAnalysis{
		OverallSQI: Round(rawPCT, 2),
		Accuracy:   acc,
		Time:       timeM,
		Difficulty: diff,
		Behavior:   beh,
		Skipping:   skip,
		Efficiency: eff,
	}
}

// helper functions for weights and normalization

func getImportanceWeight(val string) float64 {
	switch val {
	case "A":
		return 1.0
	case "B":
		return 0.7
	case "C":
		return 0.5
	default:
		return 1.0
	}
}

func getDifficultyWeight(val string) float64 {
	switch val {
	case "E":
		return 0.6
	case "M":
		return 1.0
	case "H":
		return 1.4
	default:
		return 1.0
	}
}

func getTypeWeight(val string) float64 {
	switch val {
	case "Practical":
		return 1.1
	case "Theory":
		return 1.0
	default:
		return 1.0
	}
}

func clamp(val, min, max float64) float64 {
	return math.Max(min, math.Min(max, val))
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func Round(val float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(val*pow) / pow
}
