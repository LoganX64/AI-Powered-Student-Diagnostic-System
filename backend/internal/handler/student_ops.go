package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/types"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateStudentRequest struct {
	Name        string `json:"name" binding:"required"`
	StudentCode string `json:"student_code" binding:"required"`
	CoachID     int    `json:"coach_id"`
}

func (h *AdminHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	role := c.GetString("role")

	if role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "super-admin has no access to student scores"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := resolveCoachID(c, h.CoachRepo)
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
			c.JSON(http.StatusForbidden, gin.H{"error": "student not found or not assigned to this coach"})
			return
		}
	}

	includeAnalysis := c.Query("include_analysis") == "true"
	compute := c.Query("compute") == "true"

	result, err := h.AttemptService.GetStudentSQI(services.GetStudentSQIInput{
		StudentID:       studentID,
		TenantID:        tenantID,
		IncludeAnalysis: includeAnalysis,
		Compute:         compute,
	})
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch student SQI")
		return
	}

	c.JSON(http.StatusOK, result)
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

	tenantID, err := resolveTenantID(c, h.UserRepo)
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
			"test":       gin.H{"id": testID, "title": testTitle},
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
		var payload types.DiagnosticPayloadV2
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

func (h *AdminHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := resolveTenantID(c, h.UserRepo)
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

func (h *AdminHandler) ListStudents(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	includeDeactivated := c.Query("include_deactivated") == "true"
	search := c.Query("search")

	students, total, err := h.StudentRepo.List(tenantID, coachID, includeDeactivated, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch students")
		return
	}

	if students == nil {
		students = []repository.StudentRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *AdminHandler) GetStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid
	}

	student, err := h.StudentRepo.GetDetail(studentID, tenantID, coachID)
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

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid

		exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, cid)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
			return
		}
	} else {
		exists, err := h.StudentRepo.Exists(studentID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
			return
		}
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	assignments, total, err := h.AssignmentRepo.ListByStudent(studentID, coachID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch assignments")
		return
	}

	if assignments == nil {
		assignments = []repository.AssignmentRow{}
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
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid
	}

	found, err := h.StudentRepo.SoftDelete(studentID, tenantID, userID, coachID)
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

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid
	}

	found, err := h.StudentRepo.Reactivate(studentID, tenantID, coachID)
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
