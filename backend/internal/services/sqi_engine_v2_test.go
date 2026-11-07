package services

import "testing"

func TestAnalyzeBlankAttemptsHaveNoSQIAndNoUnansweredRisk(t *testing.T) {
	tests := []struct {
		name        string
		difficulty  string
		seen        bool
		wantUnseen  int
		wantSkipped int
	}{
		{name: "empty test"},
		{name: "easy unseen", difficulty: "E", wantUnseen: 1},
		{name: "non-easy unseen", difficulty: "M", wantUnseen: 1},
		{name: "seen skipped", difficulty: "M", seen: true, wantSkipped: 1},
		{name: "hard skipped", difficulty: "H", seen: true, wantSkipped: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var questions []QuestionMetaV2
			var answers []AnswerLogV2
			if tt.name != "empty test" {
				questions = []QuestionMetaV2{{QuestionID: 1, Marks: 1, Importance: "medium", Difficulty: tt.difficulty, Type: "mcq"}}
				if tt.seen {
					answers = []AnswerLogV2{{QuestionID: 1, Seen: true}}
				}
			}

			got := Analyze(questions, answers, ExamConfigV2{})
			if got.OverallSQI != 0 {
				t.Errorf("OverallSQI = %v, want 0", got.OverallSQI)
			}
			if got.Dimensions.Mastery != 0 {
				t.Errorf("Mastery = %v, want 0", got.Dimensions.Mastery)
			}
			if got.Dimensions.Risk != 0 {
				t.Errorf("Risk = %v, want 0", got.Dimensions.Risk)
			}
			if got.ExamSummary.Unseen != tt.wantUnseen || got.ExamSummary.Skipped != tt.wantSkipped {
				t.Errorf("summary unseen/skipped = %d/%d, want %d/%d", got.ExamSummary.Unseen, got.ExamSummary.Skipped, tt.wantUnseen, tt.wantSkipped)
			}
		})
	}
}

func TestAnalyzeAttemptedAnswerProducesPerformanceScore(t *testing.T) {
	got := Analyze(
		[]QuestionMetaV2{{QuestionID: 1, Marks: 1, Importance: "medium", Difficulty: "M", Type: "mcq", ExpectedTime: 10}},
		[]AnswerLogV2{{QuestionID: 1, SelectedAnswer: "A", CorrectAnswer: "A", TimeSpent: 10, Seen: true}},
		ExamConfigV2{},
	)

	if got.Dimensions.Mastery != 100 {
		t.Errorf("Mastery = %v, want 100 for a correct-only attempt", got.Dimensions.Mastery)
	}
	if got.OverallSQI <= 0 {
		t.Errorf("OverallSQI = %v, want positive score for a correct answer", got.OverallSQI)
	}
}

func TestUnansweredQuestionDoesNotChangeAnsweredRisk(t *testing.T) {
	answered := QuestionMetaV2{QuestionID: 1, Marks: 1, Importance: "medium", Difficulty: "M", Type: "mcq", ExpectedTime: 10}
	correct := AnswerLogV2{QuestionID: 1, SelectedAnswer: "A", CorrectAnswer: "A", TimeSpent: 10, Seen: true}
	base := Analyze([]QuestionMetaV2{answered}, []AnswerLogV2{correct}, ExamConfigV2{})

	for _, blank := range []struct {
		name string
		meta QuestionMetaV2
		ans  []AnswerLogV2
	}{
		{name: "unseen", meta: QuestionMetaV2{QuestionID: 2, Importance: "medium", Difficulty: "E", Type: "mcq"}},
		{name: "skipped", meta: QuestionMetaV2{QuestionID: 2, Importance: "medium", Difficulty: "H", Type: "mcq"}, ans: []AnswerLogV2{{QuestionID: 2, Seen: true}}},
	} {
		t.Run(blank.name, func(t *testing.T) {
			withBlank := Analyze([]QuestionMetaV2{answered, blank.meta}, append([]AnswerLogV2{correct}, blank.ans...), ExamConfigV2{})
			if withBlank.Dimensions.Risk != base.Dimensions.Risk {
				t.Errorf("Risk with unanswered question = %v, want answered-only risk %v", withBlank.Dimensions.Risk, base.Dimensions.Risk)
			}
		})
	}
}
