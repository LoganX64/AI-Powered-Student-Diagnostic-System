package handlers

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CoachHandler struct {
	UserRepo       *repository.UserRepo
	StudentRepo    *repository.StudentRepo
	CoachRepo      *repository.CoachRepo
	TestRepo       *repository.TestRepo
	AssignmentRepo *repository.AssignmentRepo
	AttemptRepo    *repository.AttemptRepo
}

func NewCoachHandler(
	userRepo *repository.UserRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testRepo *repository.TestRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
) *CoachHandler {
	return &CoachHandler{
		UserRepo:       userRepo,
		StudentRepo:    studentRepo,
		CoachRepo:      coachRepo,
		TestRepo:       testRepo,
		AssignmentRepo: assignmentRepo,
		AttemptRepo:    attemptRepo,
	}
}

func (h *CoachHandler) getCoachDetailsFromUser(userID int) (int, int, error) {
	return h.CoachRepo.GetIDAndTenantFromUser(userID)
}

func (h *CoachHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	name, err := h.StudentRepo.GetName(studentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or access denied"})
		return
	}

	includeAnalysis := c.Query("include_analysis") == "true"
	compute := c.Query("compute") == "true"

	if compute {
		uncomputed, _ := h.AttemptRepo.GetUncomputedAttempts(studentID)
		for _, pair := range uncomputed {
			attemptID, testID := pair[0], pair[1]
			payload, err := calculateAttemptSQIAnalysis(h.AttemptRepo.DB, attemptID, testID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate sqi"})
				return
			}
			analysisJSON, _ := json.Marshal(payload)
			_ = h.AttemptRepo.StoreResult(attemptID, payload.OverallSQI, payload.ExamSummary.NetScore, analysisJSON, payload.Version)
		}
	}

	_ = coachID

	resultRows, _ := h.AttemptRepo.GetResults(studentID, includeAnalysis)

	var attempts []AttemptResult
	var total float64
	for _, r := range resultRows {
		attempts = append(attempts, AttemptResult{
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

	c.JSON(http.StatusOK, gin.H{
		"student_id":  studentID,
		"name":        name,
		"attempts":    attempts,
		"average_sqi": helper.Round2V2(avgSQI),
		"total_tests": len(resultRows),
	})
}

func (h *CoachHandler) GetAssignmentResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("assignmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	exists, _ := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	testID, status, assignedAt, testTitle, err := h.AssignmentRepo.GetByIDForCoach(assignmentID, studentID, coachID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	studentName, studentCode, err := h.StudentRepo.GetNameCode(studentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	attemptID, submittedAt, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
			"test":       gin.H{"id": testID, "title": testTitle},
			"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
			"attempt":    nil,
			"sqi_score":  nil,
			"answers":    []interface{}{},
		})
		return
	}

	sqiScore, analysisJSON, _ := h.AttemptRepo.GetSQIResult(attemptID)
	answers, _ := h.AttemptRepo.GetAnswerDetails(attemptID)

	var analysis interface{}
	if analysisJSON.Valid && analysisJSON.String != "" {
		var payload services.DiagnosticPayloadV2
		if err := json.Unmarshal([]byte(analysisJSON.String), &payload); err == nil {
			analysis = payload
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
		"test":       gin.H{"id": testID, "title": testTitle},
		"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
		"attempt":    gin.H{"id": attemptID, "submitted_at": submittedAt.Time},
		"sqi_score":  sqiScore.Float64,
		"analysis":   analysis,
		"answers":    answers,
	})
}

func (h *CoachHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	id, err := h.StudentRepo.Create(tenantID, req.Name, req.StudentCode, coachID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"student_id": id})
}

type CreateCoachTestRequest struct {
	Title     string  `json:"title" binding:"required"`
	SubjectID int     `json:"subject_id" binding:"required"`
	Duration  int     `json:"duration" binding:"required"`
	ExamDate  *string `json:"exam_date"`
}

func (h *CoachHandler) CreateTest(c *gin.Context) {
	var req CreateCoachTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	id, err := h.TestRepo.Create(tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test_id": id})
}

func (h *CoachHandler) CreateQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
		return
	}

	questions, err := parseQuestionRequests(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if len(questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one question is required"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	exists, _ := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id or access denied"})
		return
	}

	for i, question := range questions {
		if validationErr := repository.ValidateQuestionRequest(question); validationErr != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr, "position": i})
			return
		}
	}

	questionIDs, err := h.TestRepo.CreateQuestions(testID, questions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "failed to create questions"})
		return
	}

	response := gin.H{"question_ids": questionIDs, "count": len(questionIDs), "message": "questions created successfully"}
	if len(questionIDs) == 1 {
		response["question_id"] = questionIDs[0]
	}
	c.JSON(http.StatusCreated, response)
}

func (h *CoachHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	studentCoachID, _, err := h.StudentRepo.GetCoachIDAndTenantID(req.StudentID)
	if err != nil || studentCoachID != coachID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not owned by coach, or deactivated"})
		return
	}

	testCoachID, _, err := h.StudentRepo.GetTestCoachAndTenant(req.TestID)
	if err != nil || testCoachID != coachID {
		c.JSON(http.StatusForbidden, gin.H{"error": "test not owned by coach"})
		return
	}

	_ = tenantID

	id, err := h.AssignmentRepo.Create(req.StudentID, req.TestID, coachID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"assignment_id": id})
}

func (h *CoachHandler) ListTests(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	tests, total, err := h.TestRepo.List(tenantID, &coachID, search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *CoachHandler) ListStudents(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	includeDeactivated := c.Query("include_deactivated") == "true"

	students, total, err := h.StudentRepo.List(tenantID, &coachID, includeDeactivated, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *CoachHandler) GetStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	student, err := h.StudentRepo.GetDetail(studentID, tenantID, &coachID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	c.JSON(http.StatusOK, student)
}

func (h *CoachHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	exists, _ := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	assignments, total, err := h.AssignmentRepo.ListByStudent(studentID, &coachID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *CoachHandler) DeleteStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	found, err := h.StudentRepo.SoftDelete(studentID, tenantID, userID, &coachID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account deactivated"})
}

func (h *CoachHandler) ReactivateStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	found, err := h.StudentRepo.Reactivate(studentID, tenantID, &coachID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account reactivated"})
}

func (h *CoachHandler) ListSubjects(c *gin.Context) {
	userID := c.GetInt("user_id")
	_, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))

	subjects, total, err := h.TestRepo.ListSubjects(tenantID, "", limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": subjects})
}

func (h *CoachHandler) ListAssignments(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	testID := c.Query("test_id")

	assignments, total, err := h.AssignmentRepo.ListAll(tenantID, &coachID, testID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}
