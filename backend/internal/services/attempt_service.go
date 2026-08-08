package services

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/types"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lib/pq"
)

type AttemptService struct {
	AttemptRepo    *repository.AttemptRepo
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	TestPaperRepo       *repository.TestPaperRepo
}

func NewAttemptService(attemptRepo *repository.AttemptRepo, assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testPaperRepo *repository.TestPaperRepo) *AttemptService {
	return &AttemptService{
		AttemptRepo:    attemptRepo,
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		TestPaperRepo:       testPaperRepo,
	}
}

// ─────────────────────────────────────────────
// SubmitAnswers
// ─────────────────────────────────────────────

type AnswerInput struct {
	QuestionID        int     `json:"question_id"`
	SelectedAnswer    string  `json:"selected_answer"`
	TimeSpent         float64 `json:"time_spent"`
	Seen              *bool   `json:"seen"`
	MarkedForReview   bool    `json:"marked_for_review"`
	Revisited         bool    `json:"revisited"`
	ChangedAnswer     bool    `json:"changed_answer"`
	WasInitiallyWrong bool    `json:"was_initially_wrong"`
}

type SubmitAnswersResult struct {
	AttemptID      int     `json:"attempt_id"`
	TotalTimeSpent float64 `json:"total_time_spent"`
	TestDuration   int     `json:"test_duration"`
}

type SubmitAnswersError struct {
	Status  int
	Message string
}

func (e *SubmitAnswersError) Error() string {
	return e.Message
}

func (s *AttemptService) SubmitAnswers(assignmentID, studentID int, answers []AnswerInput) (*SubmitAnswersResult, error) {
	owner, err := s.AssignmentRepo.GetOwnerAndTest(assignmentID)
	if err != nil {
		return nil, &SubmitAnswersError{Status: 400, Message: "invalid assignment"}
	}

	if owner.OwnerID != studentID {
		return nil, &SubmitAnswersError{Status: 403, Message: "assignment does not belong to student"}
	}

	correctMap, err := s.AttemptRepo.GetCorrectAnswers(owner.TestID)
	if err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to fetch questions"}
	}

	if err := validateAnswers(answers, correctMap, owner.Duration); err != nil {
		return nil, err
	}

	result, err := s.AttemptRepo.SubmitAnswersTx(assignmentID, correctMap, toRepoAnswers(answers), func(tx *sql.Tx) error {
		return s.AssignmentRepo.MarkSubmittedTx(tx, assignmentID)
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, &SubmitAnswersError{Status: 409, Message: "assignment already submitted"}
		}
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to submit answers"}
	}

	return &SubmitAnswersResult{
		AttemptID:      result.AttemptID,
		TotalTimeSpent: helper.Round2V2(result.TotalTimeSpent),
		TestDuration:   owner.Duration,
	}, nil
}

func validateAnswers(answers []AnswerInput, correctMap map[int]string, duration int) error {
	seen := make(map[int]bool)
	var totalTime float64
	for _, ans := range answers {
		if _, ok := correctMap[ans.QuestionID]; !ok {
			return &SubmitAnswersError{Status: 400, Message: "invalid question id"}
		}
		if seen[ans.QuestionID] {
			return &SubmitAnswersError{Status: 400, Message: "duplicate question id"}
		}
		seen[ans.QuestionID] = true
		if ans.TimeSpent < 0 {
			return &SubmitAnswersError{Status: 400, Message: "time_spent cannot be negative"}
		}
		if ans.Seen != nil && !*ans.Seen && ans.SelectedAnswer != "" {
			return &SubmitAnswersError{Status: 400, Message: "not seen question cannot have selected_answer"}
		}
		if ans.SelectedAnswer != "" && ans.SelectedAnswer != "A" && ans.SelectedAnswer != "B" && ans.SelectedAnswer != "C" && ans.SelectedAnswer != "D" {
			return &SubmitAnswersError{Status: 400, Message: "selected_answer must be A/B/C/D"}
		}
		totalTime += ans.TimeSpent
		if duration > 0 && totalTime > float64(duration) {
			return &SubmitAnswersError{Status: 400, Message: "total time_spent exceeds test duration"}
		}
	}
	return nil
}

func toRepoAnswers(answers []AnswerInput) []repository.AnswerInput {
	out := make([]repository.AnswerInput, len(answers))
	for i, a := range answers {
		out[i] = repository.AnswerInput{
			QuestionID:        a.QuestionID,
			SelectedAnswer:    a.SelectedAnswer,
			TimeSpent:         a.TimeSpent,
			Seen:              a.Seen,
			MarkedForReview:   a.MarkedForReview,
			Revisited:         a.Revisited,
			ChangedAnswer:     a.ChangedAnswer,
			WasInitiallyWrong: a.WasInitiallyWrong,
		}
	}
	return out
}

// ─────────────────────────────────────────────
// GetStudentSQI
// ─────────────────────────────────────────────

type GetStudentSQIInput struct {
	StudentID       int
	TenantID        int
	IncludeAnalysis bool
	Compute         bool
}

type AttemptResultItem struct {
	AttemptID int     `json:"attempt_id"`
	TestID    int     `json:"test_id"`
	SQI       float64 `json:"sqi_score"`
}

type StudentSQIResponse struct {
	StudentID  int                 `json:"student_id"`
	Name       string              `json:"name"`
	Attempts   []AttemptResultItem `json:"attempts"`
	AverageSQI float64             `json:"average_sqi"`
	TotalTests int                 `json:"total_tests"`
}

func (s *AttemptService) GetStudentSQI(input GetStudentSQIInput) (*StudentSQIResponse, error) {
	name, err := s.StudentRepo.GetName(input.StudentID, input.TenantID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	if input.Compute {
		uncomputed, err := s.AttemptRepo.GetUncomputedAttempts(input.StudentID)
		if err != nil {
			return nil, errors.New("failed to fetch uncomputed attempts")
		}
		for _, pair := range uncomputed {
			attemptID, testID := pair[0], pair[1]
			payload, err := s.calculateAttemptSQIAnalysis(attemptID, testID)
			if err != nil {
				return nil, errors.New("failed to calculate sqi")
			}
			analysisJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, errors.New("failed to marshal analysis")
			}
			if err := s.AttemptRepo.StoreResult(attemptID, payload.OverallSQI, payload.ExamSummary.NetScore, analysisJSON, payload.Version); err != nil {
				return nil, errors.New("failed to store result")
			}
		}
	}

	resultRows, err := s.AttemptRepo.GetResults(input.StudentID, input.IncludeAnalysis)
	if err != nil {
		return nil, errors.New("failed to fetch results")
	}

	var attempts []AttemptResultItem
	var total float64
	for _, r := range resultRows {
		attempts = append(attempts, AttemptResultItem{
			AttemptID: r.AttemptID,
			TestID:    r.TestID,
			SQI:       r.SQI,
		})
		total += r.SQI
	}

	var avgSQI float64
	if len(resultRows) > 0 {
		avgSQI = total / float64(len(resultRows))
	}

	return &StudentSQIResponse{
		StudentID:  input.StudentID,
		Name:       name,
		Attempts:   attempts,
		AverageSQI: helper.Round2V2(avgSQI),
		TotalTests: len(resultRows),
	}, nil
}

func (s *AttemptService) calculateAttemptSQIAnalysis(attemptID, testID int) (types.DiagnosticPayloadV2, error) {
	questionRows, _, err := s.TestPaperRepo.ListQuestions(testID, 1000, 0)
	if err != nil {
		return types.DiagnosticPayloadV2{}, err
	}

	subjectName, _ := s.TestPaperRepo.GetSubjectName(testID)

	questions := make([]QuestionMetaV2, len(questionRows))
	hasNegMarks := false
	for i, q := range questionRows {
		questions[i] = QuestionMetaV2{
			QuestionID:   q.ID,
			Marks:        q.Marks,
			NegMarks:     q.NegMarks,
			Importance:   q.Importance,
			Difficulty:   q.Difficulty,
			Type:         q.Type,
			ExpectedTime: q.ExpectedTime,
			ConceptTag:   q.ConceptTag,
			Subject:      subjectName,
		}
		if q.NegMarks > 0 {
			hasNegMarks = true
		}
	}

	logs, err := s.AttemptRepo.GetAnswerLogsForAnalysis(attemptID)
	if err != nil {
		return types.DiagnosticPayloadV2{}, err
	}

	answers := make([]AnswerLogV2, len(logs))
	for i, l := range logs {
		answers[i] = AnswerLogV2{
			QuestionID:        l.QuestionID,
			SelectedAnswer:    l.SelectedAnswer,
			CorrectAnswer:     l.CorrectAnswer,
			TimeSpent:         l.TimeSpent,
			MarkedForReview:   l.MarkedForReview,
			Revisited:         l.Revisited,
			ChangedAnswer:     l.ChangedAnswer,
			WasInitiallyWrong: l.WasInitiallyWrong,
			Seen:              l.Seen,
		}
	}

	duration, _ := s.TestPaperRepo.GetDuration(testID)

	cfg := ExamConfigV2{
		ExamType:           "general",
		HasNegativeMarking: hasNegMarks,
		TotalDuration:      float64(duration),
	}

	return Analyze(questions, answers, cfg), nil
}
