package services

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/types"
	"encoding/json"
	"errors"
)

type AttemptService struct {
	AttemptRepo    *repository.AttemptRepo
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	TestRepo       *repository.TestRepo
}

func NewAttemptService(attemptRepo *repository.AttemptRepo, assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testRepo *repository.TestRepo) *AttemptService {
	return &AttemptService{
		AttemptRepo:    attemptRepo,
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		TestRepo:       testRepo,
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

	existsAttempt, err := s.AttemptRepo.ExistsByAssignment(assignmentID)
	if err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to check assignment status"}
	}
	if existsAttempt {
		return nil, &SubmitAnswersError{Status: 409, Message: "assignment already submitted"}
	}

	correctMap, err := s.AttemptRepo.GetCorrectAnswers(owner.TestID)
	if err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to fetch questions"}
	}

	tx, err := s.AttemptRepo.DB.Begin()
	if err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to start transaction"}
	}
	defer tx.Rollback()

	attemptID, err := s.AttemptRepo.CreateAttemptTx(tx, assignmentID)
	if err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to create attempt"}
	}

	seenQuestionIDs := make(map[int]bool)
	var totalTimeSpent float64

	for _, ans := range answers {
		_, exists := correctMap[ans.QuestionID]
		if !exists {
			return nil, &SubmitAnswersError{Status: 400, Message: "invalid question id"}
		}

		if seenQuestionIDs[ans.QuestionID] {
			return nil, &SubmitAnswersError{Status: 400, Message: "duplicate question id"}
		}
		seenQuestionIDs[ans.QuestionID] = true

		if ans.TimeSpent < 0 {
			return nil, &SubmitAnswersError{Status: 400, Message: "time_spent cannot be negative"}
		}

		answerSeen := ans.SelectedAnswer != ""
		if ans.Seen != nil {
			answerSeen = *ans.Seen
		}
		if ans.Seen != nil && !*ans.Seen && ans.SelectedAnswer != "" {
			return nil, &SubmitAnswersError{Status: 400, Message: "not seen question cannot have selected_answer"}
		}
		if ans.SelectedAnswer != "" && ans.SelectedAnswer != "A" && ans.SelectedAnswer != "B" && ans.SelectedAnswer != "C" && ans.SelectedAnswer != "D" {
			return nil, &SubmitAnswersError{Status: 400, Message: "selected_answer must be A/B/C/D"}
		}

		if !answerSeen {
			ans.TimeSpent = 0
			ans.MarkedForReview = false
			ans.Revisited = false
			ans.ChangedAnswer = false
			ans.WasInitiallyWrong = false
		}

		totalTimeSpent += ans.TimeSpent
		if owner.Duration > 0 && totalTimeSpent > float64(owner.Duration) {
			return nil, &SubmitAnswersError{
				Status:  400,
				Message: "total time_spent exceeds test duration",
			}
		}

		correctAnswer := correctMap[ans.QuestionID]
		isCorrect := answerSeen && ans.SelectedAnswer != "" && ans.SelectedAnswer == correctAnswer

		err = s.AttemptRepo.InsertAnswerLogTx(tx, attemptID, ans.QuestionID, ans.SelectedAnswer, isCorrect, ans.TimeSpent,
			ans.MarkedForReview, ans.Revisited, ans.ChangedAnswer, ans.WasInitiallyWrong, answerSeen)
		if err != nil {
			return nil, &SubmitAnswersError{Status: 500, Message: "failed to insert answer"}
		}
	}

	if err := s.AssignmentRepo.MarkSubmittedTx(tx, assignmentID); err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to mark assignment submitted"}
	}

	if err := tx.Commit(); err != nil {
		return nil, &SubmitAnswersError{Status: 500, Message: "failed to commit transaction"}
	}

	return &SubmitAnswersResult{
		AttemptID:      attemptID,
		TotalTimeSpent: helper.Round2V2(totalTimeSpent),
		TestDuration:   owner.Duration,
	}, nil
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
	questionRows, _, err := s.TestRepo.ListQuestions(testID, 1000, 0)
	if err != nil {
		return types.DiagnosticPayloadV2{}, err
	}

	subjectName, _ := s.TestRepo.GetSubjectName(testID)

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

	duration, _ := s.TestRepo.GetDuration(testID)

	cfg := ExamConfigV2{
		ExamType:           "general",
		HasNegativeMarking: hasNegMarks,
		TotalDuration:      float64(duration),
	}

	return Analyze(questions, answers, cfg), nil
}
