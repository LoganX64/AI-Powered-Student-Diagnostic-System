package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/types"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func resolveTenantID(c *gin.Context, userRepo *repository.UserRepo) (int, error) {
	return c.GetInt("tenant_id"), nil
}

func resolveCoachID(c *gin.Context, coachRepo *repository.CoachRepo) (int, error) {
	userID := c.GetInt("user_id")
	coachID, err := coachRepo.GetIDFromUser(userID)
	if err != nil {
		return 0, err
	}
	return coachID, nil
}

func resolveCoachAndTenant(c *gin.Context, coachRepo *repository.CoachRepo) (int, int, error) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := coachRepo.GetIDAndTenantFromUser(userID)
	if err != nil {
		return 0, 0, err
	}
	return coachID, tenantID, nil
}

func getStudentIDFromContext(c *gin.Context) (int, error) {
	studentIDRaw, exists := c.Get("student_id")
	if !exists {
		return 0, fmt.Errorf("unauthorized")
	}
	studentID, ok := studentIDRaw.(int)
	if !ok {
		return 0, fmt.Errorf("invalid token data")
	}
	return studentID, nil
}

// parseIDParam reads an integer path parameter, responding 400 on failure.
func parseIDParam(c *gin.Context, name string) (int, error) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil {
		utils.BadRequest(c, "invalid "+name)
		return 0, err
	}
	return id, nil
}

func verifyCoachExists(c *gin.Context, coachID int, tenantID int, coachRepo *repository.CoachRepo) bool {
	exists, err := coachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return false
	}
	if !exists {
		utils.NotFound(c, "coach not found")
		return false
	}
	return true
}

func verifyTestAccess(c *gin.Context, testID int, role string, userRepo *repository.UserRepo, coachRepo *repository.CoachRepo, testPaperRepo *repository.TestPaperRepo, tenantID int) error {
	if role == "coach" {
		userID := c.GetInt("user_id")
		coachID, err := coachRepo.GetIDFromUser(userID)
		if err != nil {
			return fmt.Errorf("coach not found")
		}
		exists, err := testPaperRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to verify test ownership: %w", err)
		}
		if !exists {
			return fmt.Errorf("test not found or not owned by you")
		}
	} else if role == "admin" {
		exists, err := testPaperRepo.Exists(testID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to verify test: %w", err)
		}
		if !exists {
			return fmt.Errorf("invalid test_id for your organization")
		}
	} else {
		return fmt.Errorf("unauthorized role")
	}
	return nil
}

func buildAssignmentResultsResponse(
	attemptRepo *repository.AttemptRepo,
	studentID, assignmentID int,
	studentName, studentCode string,
	testID int, testTitle, status, assignedAt string,
) (gin.H, error) {
	attemptID, submittedAt, err := attemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		return gin.H{
			"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
			"test":       gin.H{"id": testID, "title": testTitle},
			"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
			"attempt":    nil,
			"sqi_score":  nil,
			"answers":    []interface{}{},
		}, nil
	}

	sqiScore, analysisJSON, err := attemptRepo.GetSQIResult(attemptID)
	if err != nil {
		return nil, err
	}
	answers, err := attemptRepo.GetAnswerDetails(attemptID)
	if err != nil {
		return nil, err
	}

  var analysis interface{}
  if len(analysisJSON) > 0 {
    var payload types.DiagnosticPayloadV2
    if err := json.Unmarshal(analysisJSON, &payload); err == nil {
      analysis = payload
    }
  }

  var submittedAtVal interface{}
  if submittedAt.Valid {
    submittedAtVal = submittedAt.Time
  }
  var sqiScoreVal interface{}
  if sqiScore.Valid {
    sqiScoreVal = sqiScore.Float64
  }

  return gin.H{
    "student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
    "test":       gin.H{"id": testID, "title": testTitle},
    "assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
    "attempt":    gin.H{"id": attemptID, "submitted_at": submittedAtVal},
    "sqi_score":  sqiScoreVal,
    "analysis":   analysis,
    "answers":    answers,
  }, nil
}
