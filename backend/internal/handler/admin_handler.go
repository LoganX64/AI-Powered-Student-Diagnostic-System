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

type AdminHandler struct {
	UserRepo       *repository.UserRepo
	StudentRepo    *repository.StudentRepo
	CoachRepo      *repository.CoachRepo
	TestRepo       *repository.TestRepo
	AssignmentRepo *repository.AssignmentRepo
	AttemptRepo    *repository.AttemptRepo
}

func NewAdminHandler(
	userRepo *repository.UserRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testRepo *repository.TestRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
) *AdminHandler {
	return &AdminHandler{
		UserRepo:       userRepo,
		StudentRepo:    studentRepo,
		CoachRepo:      coachRepo,
		TestRepo:       testRepo,
		AssignmentRepo: assignmentRepo,
		AttemptRepo:    attemptRepo,
	}
}

type AttemptResult struct {
	AttemptID int             `json:"attempt_id"`
	TestID    int             `json:"test_id"`
	SQI       float64         `json:"sqi_score"`
	Analysis  json.RawMessage `json:"analysis,omitempty"`
}

type SubjectTestResult struct {
	AttemptID int                       `json:"attempt_id"`
	TestID    int                       `json:"test_id"`
	TestTitle string                    `json:"test_title"`
	SQI       float64                   `json:"sqi_score"`
	Analysis  services.DiagnosticPayloadV2 `json:"analysis"`
}

func (h *AdminHandler) getTenantID(userID int) (int, error) {
	return h.UserRepo.GetTenantID(userID)
}

func (h *AdminHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	if role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "super-admin has no access to student scores"})
		return
	}

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.AssignmentRepo.ExistsByStudent(studentID, tenantID, &coachID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify assignment")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "not assigned to this student"})
			return
		}
	}

	name, err := h.StudentRepo.GetName(studentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	includeAnalysis := c.Query("include_analysis") == "true"
	compute := c.Query("compute") == "true"

	if compute {
		uncomputed, err := h.AttemptRepo.GetUncomputedAttempts(studentID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch uncomputed attempts")
			return
		}
		for _, pair := range uncomputed {
			attemptID, testID := pair[0], pair[1]
			payload, err := calculateAttemptSQIAnalysis(h.AttemptRepo.DB, attemptID, testID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate sqi"})
				return
			}
			analysisJSON, err := json.Marshal(payload)
			if err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to marshal analysis")
				return
			}
			if err := h.AttemptRepo.StoreResult(attemptID, payload.OverallSQI, payload.ExamSummary.NetScore, analysisJSON, payload.Version); err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to store result")
				return
			}
		}
	}

	resultRows, err := h.AttemptRepo.GetResults(studentID, includeAnalysis)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch results")
		return
	}

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

func (h *AdminHandler) GetAssignmentResults(c *gin.Context) {
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
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.StudentRepo.Exists(studentID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	_, testID, status, assignedAt, testTitle, err := h.AssignmentRepo.GetByID(assignmentID, studentID)
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
			"testing":    gin.H{"id": testID, "title": testTitle},
			"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
			"attempt":    nil,
			"sqi_score":  nil,
			"answers":    []interface{}{},
		})
		return
	}

	sqiScore, analysisJSON, err := h.AttemptRepo.GetSQIResult(attemptID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch SQI result")
		return
	}
	answers, err := h.AttemptRepo.GetAnswerDetails(attemptID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch answer details")
		return
	}

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

func (h *AdminHandler) GetStudentSubjectResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	subjectID, err := strconv.Atoi(c.Param("subject_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject id"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	if role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "super-admin has no access to student scores"})
		return
	}

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	studentName, err := h.StudentRepo.GetName(studentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "not assigned to this student"})
			return
		}
	}

	var testID int
	if testIDParam := c.Query("test_id"); testIDParam != "" {
		testID, err = strconv.Atoi(testIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
			return
		}
	}

	// simplified - actual implementation would need subject queries
	response := gin.H{
		"student_id":     studentID,
		"student_name":   studentName,
		"subject_id":     subjectID,
		"results":        []interface{}{},
		"average_sqi":    0,
		"total_attempts": 0,
	}
	if testID > 0 {
		response["filter_test_id"] = testID
	}
	c.JSON(http.StatusOK, response)
}

func calculateAttemptSQIAnalysis(db interface{}, attemptID, testID int) (services.DiagnosticPayloadV2, error) {
	// Placeholder - in real implementation, this would query DB
	return services.DiagnosticPayloadV2{}, nil
}

// Student CRUD

type CreateStudentRequest struct {
	Name        string `json:"name" binding:"required"`
	StudentCode string `json:"student_code" binding:"required"`
	CoachID     int    `json:"coach_id"`
}

func (h *AdminHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var coachID int
	if role == "coach" {
		coachID, err = h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {
		if req.CoachID == 0 {
			coachID, err = h.CoachRepo.GetIDFromUser(userID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "coach_id is required, or you must create a coach profile for yourself first"})
				return
			}
		} else {
			exists, err := h.CoachRepo.Exists(req.CoachID, tenantID)
			if err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
				return
			}
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id for your organization"})
				return
			}
			coachID = req.CoachID
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	id, err := h.StudentRepo.Create(tenantID, req.Name, req.StudentCode, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create student")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"student_id": id})
}

func (h *AdminHandler) CreateSubject(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	if role != "admin" && role != "coach" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin or coach can create subjects"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	id, err := h.TestRepo.CreateSubject(tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject already exists in your organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subject_id": id})
}

// Test CRUD

type CreateTestRequest struct {
	Title     string  `json:"title" binding:"required"`
	SubjectID int     `json:"subject_id" binding:"required"`
	CoachID   int     `json:"coach_id" binding:"required"`
	Duration  int     `json:"duration" binding:"required"`
	ExamDate  *string `json:"exam_date"`
}

func (h *AdminHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var coachID int
	if role == "coach" {
		coachID, err = h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {
		exists, err := h.CoachRepo.Exists(req.CoachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id for your organization"})
			return
		}
		coachID = req.CoachID
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	id, err := h.TestRepo.Create(tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create test")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test_id": id})
}

func (h *AdminHandler) CreateQuestion(c *gin.Context) {
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

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test ownership")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		exists, err := h.TestRepo.Exists(testID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
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
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create questions")
		return
	}

	response := gin.H{"question_ids": questionIDs, "count": len(questionIDs), "message": "questions created successfully"}
	if len(questionIDs) == 1 {
		response["question_id"] = questionIDs[0]
	}
	c.JSON(http.StatusCreated, response)
}

func parseQuestionRequests(c *gin.Context) ([]repository.QuestionRequest, error) {
	body, err := c.GetRawData()
	if err != nil {
		return nil, err
	}
	var batch []repository.QuestionRequest
	if err := json.Unmarshal(body, &batch); err == nil {
		return batch, nil
	}
	var single repository.QuestionRequest
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []repository.QuestionRequest{single}, nil
}

// Assignment

type CreateAssignmentRequest struct {
	StudentID int `json:"student_id" binding:"required"`
	TestID    int `json:"test_id" binding:"required"`
	CoachID   int `json:"coach_id" binding:"required"`
}

func (h *AdminHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var coachID int
	if role == "coach" {
		var err error
		coachID, err = h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {
		coachID = req.CoachID
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin or coach can assign tests"})
		return
	}

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	studentCoachID, studentTenantID, err := h.StudentRepo.GetCoachIDAndTenantID(req.StudentID)
	if err != nil || studentCoachID != coachID || studentTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not found, deactivated, or not in your organization"})
		return
	}

	testCoachID, testTenantID, err := h.StudentRepo.GetTestCoachAndTenant(req.TestID)
	if err != nil || testCoachID != coachID || testTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not in your organization"})
		return
	}

	id, err := h.AssignmentRepo.Create(req.StudentID, req.TestID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create assignment")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"assignment_id": id})
}

// List endpoints

func (h *AdminHandler) ListTests(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	tests, total, err := h.TestRepo.List(tenantID, nil, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch tests")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) GetTest(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}
	test, err := h.TestRepo.GetDetail(testID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *AdminHandler) GetTestQuestions(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	testIDStr := c.Param("id")
	testIDInt, err := strconv.Atoi(testIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}
	exists, err := h.TestRepo.Exists(testIDInt, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	questions, total, err := h.TestRepo.ListQuestions(testIDInt, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch questions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": questions})
}

func (h *AdminHandler) ListStudents(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	includeDeactivated := c.Query("include_deactivated") == "true"

	students, total, err := h.StudentRepo.List(tenantID, nil, includeDeactivated, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch students")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *AdminHandler) GetStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	student, err := h.StudentRepo.GetDetail(studentID, tenantID, nil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	c.JSON(http.StatusOK, student)
}

func (h *AdminHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.StudentRepo.Exists(studentID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	assignments, total, err := h.AssignmentRepo.ListByStudent(studentID, nil, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch assignments")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *AdminHandler) DeleteStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.StudentRepo.SoftDelete(studentID, tenantID, userID, nil)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete student")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account deactivated"})
}

func (h *AdminHandler) ReactivateStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.StudentRepo.Reactivate(studentID, tenantID, nil)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to reactivate student")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account reactivated"})
}

func (h *AdminHandler) ListCoaches(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")
	includeDeactivated := c.Query("include_deactivated") == "true"

	coaches, total, err := h.CoachRepo.List(tenantID, search, includeDeactivated, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coaches")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": coaches})
}

func (h *AdminHandler) GetCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	coach, err := h.CoachRepo.GetDetail(coachID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	c.JSON(http.StatusOK, coach)
}

func (h *AdminHandler) DeleteCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.CoachRepo.SoftDelete(coachID, tenantID, userID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete coach")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account deactivated"})
}

func (h *AdminHandler) ReactivateCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.CoachRepo.Reactivate(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to reactivate coach")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account reactivated"})
}

func (h *AdminHandler) ListCoachTests(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.CoachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	tests, total, err := h.TestRepo.ListByCoach(coachID, tenantID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach tests")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) ListCoachStudents(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.CoachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	students, total, err := h.StudentRepo.List(tenantID, &coachID, false, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach students")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *AdminHandler) ListSubjects(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	subjects, total, err := h.TestRepo.ListSubjects(tenantID, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch subjects")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": subjects})
}

func (h *AdminHandler) ListAssignments(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	testID := c.Query("test_id")

	assignments, total, err := h.AssignmentRepo.ListAll(tenantID, nil, testID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch assignments")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *AdminHandler) UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test ownership")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		exists, err := h.TestRepo.Exists(testID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	coachTenantID, err := h.TestRepo.CoachTenantID(req.CoachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach tenant")
		return
	}
	if coachTenantID != tenantID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coach_id does not belong to your organization"})
		return
	}

	found, err := h.TestRepo.Update(testID, tenantID, req.Title, req.SubjectID, req.CoachID, req.Duration, req.ExamDate)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to update test")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test updated successfully"})
}

func (h *AdminHandler) DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test ownership")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		exists, err := h.TestRepo.Exists(testID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	found, err := h.TestRepo.Delete(testID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete test")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test deleted successfully"})
}

func (h *AdminHandler) UpdateQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	var req repository.QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if validationErr := repository.ValidateQuestionRequest(req); validationErr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test ownership")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		exists, err := h.TestRepo.Exists(testID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	found, err := h.TestRepo.UpdateQuestion(questionID, testID, req)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to update question")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question updated successfully"})
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		exists, err := h.TestRepo.ExistsOwnedByCoach(testID, coachID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test ownership")
			return
		}
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		exists, err := h.TestRepo.Exists(testID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify test")
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	found, err := h.TestRepo.DeleteQuestion(questionID, testID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete question")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question deleted successfully"})
}
