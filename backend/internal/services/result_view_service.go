package services

import (
	"database/sql"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
)

// StudentResult is the payload a student is shown for one released attempt.
//
// It deliberately carries no SQI, no rank, and no comparison with other
// students. sqi_score is a coach-facing diagnostic; exposing it would let a
// student reverse-engineer the scoring model between sittings, and nothing here
// is relative to anybody else.
type StudentResult struct {
	AttemptID   int      `json:"attempt_id"`
	AssignmentID int     `json:"assignment_id"`
	TestTitle   string   `json:"test_title"`
	StudentName string   `json:"student_name"`
	StudentCode string   `json:"student_code"`
	MarksEarned float64  `json:"marks_earned"`
	MaxMarks    float64  `json:"max_marks"`
	Percentage  float64  `json:"percentage"`
	Correct     int      `json:"correct"`
	Wrong       int      `json:"wrong"`
	Skipped     int      `json:"skipped"`
	ReleasedAt  *string  `json:"released_at"`
}

// ResultViewService serves released results to students.
type ResultViewService struct {
	AttemptRepo *repository.AttemptRepo
	TokenRepo   *repository.ResultTokenRepo
}

func NewResultViewService(attemptRepo *repository.AttemptRepo, tokenRepo *repository.ResultTokenRepo) *ResultViewService {
	return &ResultViewService{AttemptRepo: attemptRepo, TokenRepo: tokenRepo}
}

// ErrResultNotFound covers every reason a student cannot see a result:
// an unknown token, an expired one, a revoked one, and one whose release has
// somehow been withdrawn. They are deliberately the same error so the endpoint
// cannot be used to probe which of those is true.
var ErrResultNotFound = ErrResultNotComputed

// GetByToken returns the single result a token grants access to.
//
// The token is the credential: a student may open the emailed link hours later
// without being signed in, so possession is the authorisation. The repository
// additionally requires the row to be released, so the token alone is not
// sufficient if a release is ever withdrawn.
func (s *ResultViewService) GetByToken(rawToken string) (*StudentResult, error) {
	if rawToken == "" {
		return nil, ErrResultNotFound
	}
	attemptID, err := s.TokenRepo.Resolve(rawToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrResultNotFound
		}
		return nil, err
	}

	res, err := s.AttemptRepo.GetReleasedResultByToken(utils.HashSHA256Hex(rawToken))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrResultNotFound
		}
		return nil, err
	}
	if res.AttemptID != attemptID {
		// Resolve and the result lookup must agree; they query the same row, so
		// a mismatch means something is inconsistent and nothing should be
		// served.
		return nil, ErrResultNotFound
	}
	return studentResultFrom(res), nil
}

// ListForStudent returns every released result for the signed-in student.
//
// The authenticated counterpart to GetByToken: the email link is a convenience,
// not the only way in. Unreleased results are excluded, so a student who has
// submitted but whose coach has not published sees an empty list rather than a
// withheld score.
func (s *ResultViewService) ListForStudent(studentID, tenantID int) ([]StudentResult, error) {
	rows, err := s.AttemptRepo.ListReleasedResultsForStudent(studentID, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]StudentResult, 0, len(rows))
	for i := range rows {
		out = append(out, *studentResultFrom(&rows[i]))
	}
	return out, nil
}

func studentResultFrom(r *repository.ReleasableResult) *StudentResult {
	out := &StudentResult{
		AttemptID:   r.AttemptID,
		AssignmentID: r.AssignmentID,
		TestTitle:   r.TestTitle,
		StudentName: r.StudentName,
		StudentCode: r.StudentCode,
		MarksEarned: r.RawScore,
		MaxMarks:    r.MaxScore,
		Percentage:  r.Percent,
		Correct:     r.Correct,
		Wrong:       r.Wrong,
		Skipped:     r.Skipped,
	}
	if r.ReleasedAt.Valid {
		ts := r.ReleasedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		out.ReleasedAt = &ts
	}
	return out
}