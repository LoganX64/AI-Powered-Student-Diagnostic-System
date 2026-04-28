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
		OverallSQI: round(rawPCT, 2),
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

func round(val float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(val*pow) / pow
}
