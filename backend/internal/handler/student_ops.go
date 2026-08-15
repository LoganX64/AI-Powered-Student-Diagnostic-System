package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateStudentRequest struct {
	Name        string `json:"name" binding:"required"`
	StudentCode string `json:"student_code"`
	CoachID     int    `json:"coach_id"`
	BatchID     *int   `json:"batch_id"`
}

func (h *AdminHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	role := c.GetString("role")

	if role == "super_admin" {
		utils.Forbidden(c, "super-admin has no access to student scores")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if role == "coach" {
		coachID, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
		exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			utils.Forbidden(c, "student not found or not assigned to this coach")
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

type StudentSQIBatchRequest struct {
	StudentIDs []int `json:"student_ids" binding:"required"`
}

const maxBatchStudentIDs = 500

func (h *AdminHandler) GetStudentSQIBatch(c *gin.Context) {
	role := c.GetString("role")

	if role == "super_admin" {
		utils.Forbidden(c, "super-admin has no access to student scores")
		return
	}

	var req StudentSQIBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if len(req.StudentIDs) == 0 {
		utils.BadRequest(c, "student_ids must not be empty")
		return
	}
	if len(req.StudentIDs) > maxBatchStudentIDs {
		utils.BadRequest(c, "too many student_ids")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	coachID := 0
	if role == "coach" {
		coachID, err = resolveCoachID(c, h.CoachRepo)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
	}

	result, err := h.AttemptService.GetStudentSQIBatch(req.StudentIDs, tenantID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch student SQI")
		return
	}

	if result == nil {
		result = []repository.StudentSQIMetric{}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *AdminHandler) GetAssignmentResults(c *gin.Context) {
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

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	exists, err := h.StudentRepo.Exists(studentID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		utils.NotFound(c, "student not found")
		return
	}

	_, testID, status, assignedAt, testTitle, err := h.AssignmentRepo.GetByID(assignmentID, studentID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return
	}

	studentName, studentCode, err := h.StudentRepo.GetNameCode(studentID, tenantID)
	if err != nil {
		utils.NotFound(c, "student not found")
		return
	}

	resp, err := buildAssignmentResultsResponse(
		h.AttemptRepo, studentID, assignmentID,
		studentName, studentCode,
		testID, testTitle, status, assignedAt,
	)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch results")
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AdminHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	tenantID, err := resolveTenantID(c, h.UserRepo)
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
		if req.CoachID == 0 {
			coachID, err = h.CoachRepo.GetIDFromUser(userID)
			if err != nil {
				utils.BadRequest(c, "coach_id is required, or you must create a coach profile for yourself first")
				return
			}
		} else {
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
		}
	} else {
		utils.Forbidden(c, "unauthorized role")
		return
	}

	id, code, err := ensureStudentCode(h.StudentRepo, tenantID, req.Name, req.StudentCode, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create student")
		return
	}

	if req.BatchID != nil {
		ok, err := h.BatchRepo.Exists(tenantID, *req.BatchID)
		if err == nil && ok {
			_ = h.BatchRepo.SetStudentBatch(tenantID, id, req.BatchID)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"student_id": id, "student_code": code})
}

func (h *AdminHandler) ListStudents(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
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
		utils.BadRequest(c, "invalid student id")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
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

	student, err := h.StudentRepo.GetDetail(studentID, tenantID, coachID)
	if err != nil {
		utils.NotFound(c, "student not found")
		return
	}

	c.JSON(http.StatusOK, student)
}

func (h *AdminHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
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

		exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, cid)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			utils.NotFound(c, "student not found")
			return
		}
	} else {
		exists, err := h.StudentRepo.Exists(studentID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
			return
		}
		if !exists {
			utils.NotFound(c, "student not found")
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
		utils.BadRequest(c, "invalid student id")
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := resolveTenantID(c, h.UserRepo)
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

	found, err := h.StudentRepo.SoftDelete(studentID, tenantID, userID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete student")
		return
	}
	if !found {
		utils.NotFound(c, "student not found or already deactivated")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account deactivated"})
}

func (h *AdminHandler) ReactivateStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
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

	found, err := h.StudentRepo.Reactivate(studentID, tenantID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to reactivate student")
		return
	}
	if !found {
		utils.NotFound(c, "student not found or already active")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account reactivated"})
}
