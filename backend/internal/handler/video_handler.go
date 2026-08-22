package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	AttemptRepo    *repository.AttemptRepo
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	CoachRepo      *repository.CoachRepo
	Storage        storage.Storage
}

func NewVideoHandler(
	attemptRepo *repository.AttemptRepo,
	assignmentRepo *repository.AssignmentRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	storageBackend storage.Storage,
) *VideoHandler {
	return &VideoHandler{
		AttemptRepo:    attemptRepo,
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		CoachRepo:      coachRepo,
		Storage:        storageBackend,
	}
}

func (h *VideoHandler) resolveOwnership(c *gin.Context, assignmentID int, role string) (int, error) {
	owner, err := h.AssignmentRepo.GetOwnerAndTest(assignmentID)
	if err != nil {
		return 0, fmt.Errorf("assignment not found")
	}

	studentID := owner.OwnerID
	studentCoachID, studentTenantID, err := h.StudentRepo.GetCoachIDAndTenantID(studentID)
	if err != nil {
		return 0, fmt.Errorf("student not found")
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID != studentTenantID {
		return 0, fmt.Errorf("not in your organization")
	}

	if role == "coach" {
		viewerCoachID, err := h.CoachRepo.GetIDFromUser(c.GetInt("user_id"))
		if err != nil {
			return 0, fmt.Errorf("coach profile not found")
		}
		if viewerCoachID != studentCoachID {
			return 0, fmt.Errorf("student not assigned to you")
		}
	}

	return studentID, nil
}

func (h *VideoHandler) ListVideoChunks(c *gin.Context) {
	role, ok := getRoleFromContext(c)
	if !ok {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	if role != "admin" && role != "coach" {
		utils.Unauthorized(c, "admin or coach role required")
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment_id")
		return
	}

	if _, err := h.resolveOwnership(c, assignmentID, role); err != nil {
		utils.Forbidden(c, err.Error())
		return
	}

	attemptID, _, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		utils.NotFound(c, "no attempt found for this assignment")
		return
	}

	prefix := fmt.Sprintf("video/%d/", attemptID)
	keys, err := h.Storage.List(c.Request.Context(), prefix)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to list video chunks")
		return
	}

	chunks := make([]string, 0, len(keys))
	for _, key := range keys {
		idx := strings.TrimPrefix(key, prefix)
		idx = strings.TrimSuffix(idx, ".webm")
		if idx != "" {
			chunks = append(chunks, idx)
		}
	}
	sort.Strings(chunks)

	c.JSON(http.StatusOK, gin.H{
		"assignment_id": assignmentID,
		"attempt_id":    attemptID,
		"chunks":        chunks,
	})
}

func (h *VideoHandler) StreamVideoChunk(c *gin.Context) {
	role, ok := getRoleFromContext(c)
	if !ok {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	if role != "admin" && role != "coach" {
		utils.Unauthorized(c, "admin or coach role required")
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment_id")
		return
	}

	index := c.Param("index")
	if index == "" {
		utils.BadRequest(c, "missing chunk index")
		return
	}

	if _, err := h.resolveOwnership(c, assignmentID, role); err != nil {
		utils.Forbidden(c, err.Error())
		return
	}

	attemptID, _, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		utils.NotFound(c, "no attempt found for this assignment")
		return
	}

	key := fmt.Sprintf("video/%d/%s.webm", attemptID, index)
	rc, err := h.Storage.Get(c.Request.Context(), key)
	if err != nil {
		utils.NotFound(c, "video chunk not found")
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "video/webm")
	c.Header("Cache-Control", "public, max-age=3600")
	if _, err := io.Copy(c.Writer, rc); err != nil {
		log.Printf("[VIDEO] Stream error for %s: %v", key, err)
	}
}

func getRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
