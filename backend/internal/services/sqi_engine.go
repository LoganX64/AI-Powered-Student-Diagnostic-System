package services

import (
	"ai-student-diagnostic/backend/internal/helper"
	"sort"
)

type QuestionMeta struct {
	QuestionID   int
	Marks        float64
	NegMarks     float64
	Importance   string
	Difficulty   string
	Type         string
	ExpectedTime float64
	ConceptTag   string
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
	totalWeighted, maxPossible := calculateSQITotals(questions, answers)
	rawPCT := normalizeSQI(totalWeighted, maxPossible)

	return SQIResult{
		OverallSQI: helper.Round(rawPCT, 2),
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

	ConceptSQI        map[string]float64
	SummaryPriorities []SummaryPriority
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

type SummaryPriority struct {
	ConceptTag        string  `json:"concept_tag"`
	PriorityScore     float64 `json:"priority_score"`
	WrongAtLeastOnce  bool    `json:"wrong_at_least_once"`
	ImportanceScore   float64 `json:"importance_score"`
	TimeProxyScore    float64 `json:"time_proxy_score"`
	DiagnosticQuality float64 `json:"diagnostic_quality"`
	ConceptSQI        float64 `json:"concept_sqi"`
}

type conceptAggregate struct {
	Questions       []QuestionMeta
	Answers         []AnswerLog
	WrongAtLeastOne bool
	ImportanceSum   float64
	TimeProxySum    float64
	AnsweredCount   int
}

func CalculateSQIAnalysis(questions []QuestionMeta, answers []AnswerLog) SQIAnalysis {
	answerMap := make(map[int]AnswerLog)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	var totalWeighted, maxPossible float64

	var acc AccuracyMetrics
	var timeM TimeMetrics
	var beh BehaviorMetrics
	var skip SkippingMetrics
	var eff EfficiencyMetrics
	var diff DifficultyMetrics

	acc.TotalQuestions = len(questions)
	concepts := make(map[string]*conceptAggregate)

	for _, q := range questions {

		ans, attempted := answerMap[q.QuestionID]
		weighted, questionMax, isCorrect, timeRatio := calculateQuestionWeighted(q, ans, attempted)
		totalWeighted += weighted
		maxPossible += questionMax

		conceptTag := q.ConceptTag
		if conceptTag == "" {
			conceptTag = "uncategorized"
		}
		concept := concepts[conceptTag]
		if concept == nil {
			concept = &conceptAggregate{}
			concepts[conceptTag] = concept
		}
		concept.Questions = append(concept.Questions, q)
		concept.ImportanceSum += helper.GetImportanceWeight(q.Importance)

		if !attempted {
			skip.NotSeen++
			acc.Skipped++
			concept.WrongAtLeastOne = true
			continue
		}

		concept.Answers = append(concept.Answers, ans)

		if ans.Seen && ans.SelectedAnswer == "" {
			skip.SeenNotAnswered++
			acc.Skipped++
			concept.WrongAtLeastOne = true
			continue
		}

		skip.Answered++
		acc.Attempted++

		if isCorrect {
			acc.Correct++
		} else {
			acc.Wrong++
			concept.WrongAtLeastOne = true
		}

		timeM.TotalTime += ans.TimeSpent
		timeM.TotalExpectedTime += q.ExpectedTime
		concept.TimeProxySum += getTimeProxyScore(timeRatio)
		concept.AnsweredCount++

		if timeRatio > 2 {
			timeM.VerySlowCount++
		} else if timeRatio > 1.5 {
			timeM.SlowCount++
		} else if timeRatio < 0.5 {
			timeM.FastCount++
		}

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
	}

	rawPCT := normalizeSQI(totalWeighted, maxPossible)

	if acc.TotalQuestions > 0 {
		timeM.AvgTimePerQuestion = timeM.TotalTime / float64(acc.TotalQuestions)
		timeM.ExpectedAvgTime = timeM.TotalExpectedTime / float64(acc.TotalQuestions)
	}
	if acc.Attempted > 0 {
		acc.AccuracyPct = helper.Round(float64(acc.Correct)/float64(acc.Attempted)*100, 2)
	}

	conceptSQI, summaryPriorities := calculateConceptOutputs(concepts)

	return SQIAnalysis{
		OverallSQI:        helper.Round(rawPCT, 2),
		Accuracy:          acc,
		Time:              timeM,
		Difficulty:        diff,
		Behavior:          beh,
		Skipping:          skip,
		Efficiency:        eff,
		ConceptSQI:        conceptSQI,
		SummaryPriorities: summaryPriorities,
	}
}

func calculateSQITotals(questions []QuestionMeta, answers []AnswerLog) (float64, float64) {
	answerMap := make(map[int]AnswerLog)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	var totalWeighted float64
	var maxPossible float64

	for _, q := range questions {
		ans, attempted := answerMap[q.QuestionID]
		weighted, questionMax, _, _ := calculateQuestionWeighted(q, ans, attempted)
		totalWeighted += weighted
		maxPossible += questionMax
	}

	return totalWeighted, maxPossible
}

func calculateQuestionWeighted(q QuestionMeta, ans AnswerLog, attempted bool) (float64, float64, bool, float64) {
	weightFactor := helper.GetImportanceWeight(q.Importance) *
		helper.GetDifficultyWeight(q.Difficulty) *
		helper.GetTypeWeight(q.Type)

	maxPossible := q.Marks * weightFactor
	if !attempted || ans.SelectedAnswer == "" {
		return -q.NegMarks * weightFactor, maxPossible, false, 0
	}

	isCorrect := ans.SelectedAnswer == ans.CorrectAnswer
	base := -q.NegMarks
	if isCorrect {
		base = q.Marks
	}

	weighted := base * weightFactor
	timeRatio := helper.SafeDivide(ans.TimeSpent, q.ExpectedTime)

	if timeRatio > 2 {
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

	return weighted, maxPossible, isCorrect, timeRatio
}

func normalizeSQI(totalWeighted float64, maxPossible float64) float64 {
	if maxPossible <= 0 {
		return 0
	}

	return helper.Clamp(totalWeighted/maxPossible*100, 0, 100)
}

func calculateConceptOutputs(concepts map[string]*conceptAggregate) (map[string]float64, []SummaryPriority) {
	conceptSQI := make(map[string]float64, len(concepts))
	priorities := make([]SummaryPriority, 0, len(concepts))

	for tag, concept := range concepts {
		score := CalculateSQI(concept.Questions, concept.Answers).OverallSQI
		conceptSQI[tag] = score

		questionCount := len(concept.Questions)
		if questionCount == 0 {
			continue
		}

		wrongScore := 0.0
		if concept.WrongAtLeastOne {
			wrongScore = 1.0
		}

		importanceScore := concept.ImportanceSum / float64(questionCount)
		timeProxyScore := 0.7
		if concept.AnsweredCount > 0 {
			timeProxyScore = concept.TimeProxySum / float64(concept.AnsweredCount)
		}
		diagnosticQuality := 1 - score/100

		priorityScore := 0.40*wrongScore +
			0.25*importanceScore +
			0.20*timeProxyScore +
			0.15*diagnosticQuality

		priorities = append(priorities, SummaryPriority{
			ConceptTag:        tag,
			PriorityScore:     helper.Round(helper.Clamp(priorityScore, 0, 1), 4),
			WrongAtLeastOnce:  concept.WrongAtLeastOne,
			ImportanceScore:   helper.Round(importanceScore, 4),
			TimeProxyScore:    helper.Round(timeProxyScore, 4),
			DiagnosticQuality: helper.Round(diagnosticQuality, 4),
			ConceptSQI:        score,
		})
	}

	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].PriorityScore > priorities[j].PriorityScore
	})

	return conceptSQI, priorities
}

func getTimeProxyScore(timeRatio float64) float64 {
	switch {
	case timeRatio > 1.5:
		return 0.4
	case timeRatio < 0.5:
		return 1.0
	default:
		return 0.7
	}
}
