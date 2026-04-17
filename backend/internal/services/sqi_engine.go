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
}

type SQIResult struct {
	OverallSQI float64
}

func CalculateSQI(question []QuestionMeta, answers []AnswerLog) SQIResult {
	qMap := make(map[int]QuestionMeta)
	for _, q := range question {
		qMap[q.QuestionID] = q
	}

	var totalWeighted float64
	var maxPossible float64

	for _, ans := range answers {
		q, exists := qMap[ans.QuestionID]
		if !exists {
			continue
		}

		// base score
		isCorrect := ans.SelectedAnswer == ans.CorrectAnswer
		var base float64
		if isCorrect {
			base = q.Marks
		} else {
			base = -q.NegMarks
		}

		// weights
		importanceW := getImportanceWeight(q.Importance)
		difficultyW := getDifficultyWeight(q.Difficulty)
		typeW := getTypeWeight(q.Type)

		weighted := base * importanceW * difficultyW * typeW

		// behavioral adjustments
		timeRatio := safeDivide(ans.TimeSpent, q.ExpectedTime)
		if timeRatio > 2.0 {
			weighted *= 0.8
		} else if timeRatio > 1.5 {
			weighted *= 0.9
		}

		if ans.MarkedForReview && !isCorrect {
			weighted *= 0.9
		}
		if ans.Revisited && ans.ChangedAnswer && isCorrect && ans.WasInitiallyWrong {
			weighted += 0.2 * q.Marks
		}

		totalWeighted += weighted

		// mass possible score
		maxPossible += q.Marks * importanceW * difficultyW * typeW
	}
	// normalize
	rawPCT := 0.0
	if maxPossible > 0 {
		rawPCT = totalWeighted / maxPossible * 100
	}
	rawPCT = clamp(rawPCT, 0, 100)

	return SQIResult{
		OverallSQI: round(rawPCT, 2),
	}
}

// Helper functions for weights, safe division, clamping, and rounding would be defined here.

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
