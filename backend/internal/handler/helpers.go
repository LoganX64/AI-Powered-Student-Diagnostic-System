package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func resolveTenantID(c *gin.Context, userRepo *repository.UserRepo) (int, error) {
	userID := c.GetInt("user_id")
	tenantID, err := userRepo.GetTenantID(userID)
	if err != nil {
		return 0, err
	}
	return tenantID, nil
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

func verifyCoachExists(c *gin.Context, coachID int, tenantID int, coachRepo *repository.CoachRepo) bool {
	exists, err := coachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return false
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
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
