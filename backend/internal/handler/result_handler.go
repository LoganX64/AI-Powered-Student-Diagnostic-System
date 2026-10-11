package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"github.com/gin-gonic/gin"
)

// ResultHandler serves the coach/admin release action and the student's own
// view of a released result.
type ResultHandler struct {
	ReleaseService *services.ResultReleaseService
	AttemptRepo    *repository.AttemptRepo
	TokenRepo      *repository.ResultTokenRepo
	StudentRepo    *repository.StudentRepo
	CoachRepo      *repository.CoachRepo
}

func NewResultHandler(
	releaseService *services.ResultReleaseService,
	attemptRepo *repository.AttemptRepo,
	tokenRepo *repository.ResultTokenRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
) *ResultHandler {
	return &ResultHandler{
		ReleaseService: releaseService,
		AttemptRepo:    attemptRepo,
		TokenRepo:      tokenRepo,
		StudentRepo:    studentRepo,
		CoachRepo:      coachRepo,
	}
}

// POST /admin|coach/students/:id/assignments/:assignmentId/release-results
//
// Releases one student's result for one assignment. Per attempt: a retest is a
// separate assignment and is released independently.
func (h *ResultHandler) ReleaseAssignmentResult(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}
	assignmentID, err := strconv.Atoi(c.Param("assignmentId"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	// Ownership check before anything else. The admin owns the whole tenant; a
	// coach must actually be assigned to this student.
	tenantID := c.GetInt("tenant_id")
	if c.GetString("role") != "admin" {
		coachID, _, err := resolveCoachAndTenant(c, h.CoachRepo)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
		owns, err := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !owns {
			// 404 rather than 403: a coach should not learn that a student id
			// exists in the org but is not theirs.
			utils.NotFound(c, "student not found")
			return
		}
	}

	outcome, err := h.ReleaseService.ReleaseResult(assignmentID, c.GetInt("user_id"))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrResultNotComputed):
			// Distinct from 404: the result exists as a concept but the SQI job
			// has not run, and the coach needs to know to run it.
			utils.BadRequest(c, "result has not been computed yet — run the SQI computation first")
		case errors.Is(err, services.ErrResultAlreadyReleased):
			c.JSON(http.StatusOK, gin.H{"message": "result is already released"})
		default:
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to release result")
		}
		return
	}

// A repeated release changes nothing, so say so rather than reporting a
	// fresh release — otherwise a coach cannot tell the two apart.
	message := "result released"
	if outcome.AlreadyRel {
		message = "result is already released"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          message,
		"attempt_id":       outcome.AttemptID,
		"already_released": outcome.AlreadyRel,
		"email_sent":       outcome.EmailSent,
		// Returned so a coach can copy the link when the student has no email on
		// file or SMTP is unconfigured.
		"link": outcome.Link,
	})
}