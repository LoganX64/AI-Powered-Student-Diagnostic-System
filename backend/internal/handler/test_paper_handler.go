package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateTestRequest struct {
	Title       string  `json:"title" binding:"required"`
	SubjectID   int     `json:"subject_id" binding:"required"`
	SubjectName string  `json:"subject_name" binding:"required"`
	CoachID     int     `json:"coach_id"`
	Duration    int     `json:"duration" binding:"required"`
	ExamDate    *string `json:"exam_date"`
}

func (h *AdminHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	var coachID int
	if role == "coach" {
		coachID, err = h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
	} else if role == "admin" {
		exists, err := h.CoachRepo.Exists(req.CoachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
			return
		}
		if !exists {
			utils.BadRequest(c, "invalid coach_id for your organization")
			return
		}
		coachID = req.CoachID
	} else {
		utils.Forbidden(c, "unauthorized role")
		return
	}

	id, err := h.TestPaperRepo.Create(tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate, req.SubjectName)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create test")
		return
	}

	if h.QuotaMW != nil {
		h.QuotaMW.Invalidate(tenantID)
	}

	if h.NotificationService != nil {
		if err := h.NotificationService.NotifyCoachActivity(tenantID, coachID, "created test", req.Title); err != nil {
			log.Printf("[NOTIFICATION] coach activity notify failed: %v", err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"test_id": id})
}

func (h *AdminHandler) UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}

	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if err := verifyTestAccess(c, testID, role, h.UserRepo, h.CoachRepo, h.TestPaperRepo, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "test access verification failed")
		return
	}

	coachTenantID, err := h.TestPaperRepo.CoachTenantID(req.CoachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach tenant")
		return
	}
	if coachTenantID != tenantID {
		utils.BadRequest(c, "coach_id does not belong to your organization")
		return
	}

	found, err := h.TestPaperRepo.Update(testID, tenantID, req.Title, req.SubjectID, req.CoachID, req.Duration, req.ExamDate, req.SubjectName)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to update test")
		return
	}
	if !found {
		utils.NotFound(c, "test not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test updated successfully"})
}

func (h *AdminHandler) DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}

	role := c.GetString("role")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if err := verifyTestAccess(c, testID, role, h.UserRepo, h.CoachRepo, h.TestPaperRepo, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "test access verification failed")
		return
	}

	userID := c.GetInt("user_id")

	found, err := h.TestPaperRepo.Delete(testID, tenantID, userID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to deactivate test")
		return
	}
	if !found {
		utils.NotFound(c, "test not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test deactivated"})
}

func (h *AdminHandler) ListTests(c *gin.Context) {
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
		coachID = &cid
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	tests, total, err := h.TestPaperRepo.List(tenantID, coachID, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch tests")
		return
	}

	if tests == nil {
		tests = []repository.TestRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) GetTest(c *gin.Context) {
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}
	test, err := h.TestPaperRepo.GetDetail(testID, tenantID)
	if err != nil {
		utils.NotFound(c, "test not found")
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *AdminHandler) GetTestQuestions(c *gin.Context) {
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	testIDStr := c.Param("id")
	testIDInt, err := strconv.Atoi(testIDStr)
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}
	exists, err := h.TestPaperRepo.Exists(testIDInt, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
		return
	}
	if !exists {
		utils.NotFound(c, "test not found")
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	questions, total, err := h.TestPaperRepo.ListQuestions(testIDInt, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch questions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": questions})
}

func (h *AdminHandler) CreateQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test_id")
		return
	}

	var questions []repository.QuestionRequest
	if err := c.ShouldBindJSON(&questions); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if len(questions) == 0 {
		utils.BadRequest(c, "at least one question is required")
		return
	}

	role := c.GetString("role")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if err := verifyTestAccess(c, testID, role, h.UserRepo, h.CoachRepo, h.TestPaperRepo, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusForbidden, err, "test access verification failed")
		return
	}

	for i, question := range questions {
		if validationErr := repository.ValidateQuestionRequest(question); validationErr != "" {
			utils.BadRequest(c, validationErr, gin.H{"position": i})
			return
		}
	}

	currentCount, err := h.TestPaperRepo.CountQuestions(testID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to count questions")
		return
	}
	if currentCount+len(questions) > 1000 {
		utils.BadRequest(c, fmt.Sprintf("test has %d questions, adding %d would exceed limit of 1000", currentCount, len(questions)))
		return
	}

	questionIDs, err := h.TestPaperRepo.CreateQuestions(testID, questions)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create questions")
		return
	}

	response := gin.H{"question_ids": questionIDs, "count": len(questionIDs), "message": "questions created successfully"}
	if len(questionIDs) == 1 {
		response["question_id"] = questionIDs[0]
	}
	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) UpdateQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		utils.BadRequest(c, "invalid question id")
		return
	}

	var req repository.QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	if validationErr := repository.ValidateQuestionRequest(req); validationErr != "" {
		utils.BadRequest(c, validationErr)
		return
	}

	role := c.GetString("role")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if err := verifyTestAccess(c, testID, role, h.UserRepo, h.CoachRepo, h.TestPaperRepo, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "test access verification failed")
		return
	}

	found, err := h.TestPaperRepo.UpdateQuestion(questionID, testID, req)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to update question")
		return
	}
	if !found {
		utils.NotFound(c, "question not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question updated successfully"})
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid test id")
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		utils.BadRequest(c, "invalid question id")
		return
	}

	role := c.GetString("role")

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if err := verifyTestAccess(c, testID, role, h.UserRepo, h.CoachRepo, h.TestPaperRepo, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "test access verification failed")
		return
	}

	found, err := h.TestPaperRepo.DeleteQuestion(questionID, testID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete question")
		return
	}
	if !found {
		utils.NotFound(c, "question not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question deleted successfully"})
}


