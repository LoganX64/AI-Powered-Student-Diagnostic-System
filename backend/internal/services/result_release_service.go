package services

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"ai-student-diagnostic/backend/internal/repository"
)

var (
	// ErrResultNotComputed means no attempt_results row exists for the attempt,
	// so the SQI job has not run for it. The coach must compute before release.
	ErrResultNotComputed = errors.New("result has not been computed yet")

	// ErrResultAlreadyReleased means the attempt's result already carries a
	// released_at. Re-release is a no-op so a double-click cannot invalidate a
	// link the student already holds.
	ErrResultAlreadyReleased = errors.New("result is already released")
)

// ResultReleaseService publishes a computed attempt result to its student.
type ResultReleaseService struct {
	AttemptRepo  *repository.AttemptRepo
	TokenRepo    *repository.ResultTokenRepo
	Mailer       *Mailer
	TokenTTL     time.Duration
}

func NewResultReleaseService(
	attemptRepo *repository.AttemptRepo,
	tokenRepo *repository.ResultTokenRepo,
	mailer *Mailer,
) *ResultReleaseService {
	return &ResultReleaseService{
		AttemptRepo: attemptRepo,
		TokenRepo:   tokenRepo,
		Mailer:      mailer,
		// Same constant the email states, so the student is never told a
		// different window than the token actually gets.
		TokenTTL: ResultLinkValidDays * 24 * time.Hour,
	}
}

// ReleaseResult publishes one student's result for one assignment.
//
// Order matters: the release is recorded before the token is minted, so a token
// never exists for a result that was not released. Email is last and
// fire-and-forget, because a result that has been released must not appear
// unreleased just because the mail server was down.
//
// The returned link is surfaced to the coach so it can be copied manually when
// the student has no email on file or SMTP is unconfigured.
func (s *ResultReleaseService) ReleaseResult(assignmentID, releasedBy int) (*ResultReleaseOutcome, error) {
	attemptID, err := s.AttemptRepo.GetSubmittedAttemptID(assignmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrResultNotComputed
		}
		return nil, err
	}

	res, err := s.AttemptRepo.GetReleasableResult(attemptID, assignmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrResultNotComputed
		}
		return nil, err
	}
	if !res.HasResult {
		// The SQI job has not run for this attempt, so there is nothing to
		// publish. This is distinct from a computed result of zero.
		return nil, ErrResultNotComputed
	}
	if res.ReleasedAt.Valid {
		return &ResultReleaseOutcome{
			AttemptID:  res.AttemptID,
			StudentID:  res.StudentID,
			TestTitle:  res.TestTitle,
			AlreadyRel: true,
		}, nil
	}

	released, err := s.AttemptRepo.MarkResultReleased(res.AttemptID, releasedBy)
	if err != nil {
		return nil, err
	}
	if !released {
		// Lost a race with a concurrent release; treat it as already done rather
		// than minting a second token.
		return &ResultReleaseOutcome{AttemptID: res.AttemptID, StudentID: res.StudentID, AlreadyRel: true}, nil
	}

	token, err := s.TokenRepo.Mint(res.AttemptID, s.TokenTTL)
	if err != nil {
		// The release is already recorded, so the result is visible to the
		// student via the authenticated route even though the link failed.
		log.Printf("[RESULT] mint token for attempt %d failed after release: %v", res.AttemptID, err)
		return &ResultReleaseOutcome{AttemptID: res.AttemptID, StudentID: res.StudentID, TokenMinted: false}, nil
	}

	link := s.Mailer.resultLink(token)
	out := &ResultReleaseOutcome{
		AttemptID:   res.AttemptID,
		StudentID:   res.StudentID,
		StudentName: res.StudentName,
		TestTitle:   res.TestTitle,
		Link:        link,
		TokenMinted: true,
	}

	if res.StudentEmail.Valid && res.StudentEmail.String != "" {
		if err := s.Mailer.SendResultReleased(
			res.StudentEmail.String, res.StudentName, res.TestTitle, token,
			res.RawScore, res.MaxScore, res.Percent,
		); err != nil {
			// Fire-and-forget: the link is returned to the coach regardless.
			log.Printf("[RESULT] send release email to %s failed: %v", res.StudentEmail.String, err)
			out.EmailSent = false
		} else {
			out.EmailSent = true
		}
	} else {
		log.Printf("[RESULT] student %d has no email on file; release link: %s", res.StudentID, link)
		out.EmailSent = false
	}

	return out, nil
}

// ResultReleaseOutcome describes what a release did, so the handler can tell a
// coach whether mail was sent or they need to copy the link themselves.
type ResultReleaseOutcome struct {
	AttemptID   int
	StudentID   int
	StudentName string
	TestTitle   string
	Link        string
	EmailSent   bool
	TokenMinted bool
	AlreadyRel  bool
}