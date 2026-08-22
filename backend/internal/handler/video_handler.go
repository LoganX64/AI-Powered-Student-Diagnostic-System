package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	hasVP, vpErr := h.hasVideoProctoring(assignmentID)
	if vpErr != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, vpErr, "failed to check policy")
		return
	}
	if !hasVP {
		c.JSON(http.StatusOK, gin.H{
			"assignment_id": assignmentID,
			"attempt_id":    0,
			"chunks":        []string{},
			"has_merged":    false,
		})
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
	hasMerged := false
	for _, key := range keys {
		idx := strings.TrimPrefix(key, prefix)
		idx = strings.TrimSuffix(idx, ".webm")
		if idx == "merged" {
			hasMerged = true
		} else if idx != "" {
			chunks = append(chunks, idx)
		}
	}
	sort.Strings(chunks)

	c.JSON(http.StatusOK, gin.H{
		"assignment_id": assignmentID,
		"attempt_id":    attemptID,
		"chunks":        chunks,
		"has_merged":    hasMerged,
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

// StreamMergedVideo serves the merged video. Merges on first request if not already done.
func (h *VideoHandler) StreamMergedVideo(c *gin.Context) {
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

	mergedKey := fmt.Sprintf("video/%d/merged.webm", attemptID)

	// Try to serve existing merged file.
	rc, err := h.Storage.Get(c.Request.Context(), mergedKey)
	if err == nil {
		defer rc.Close()
		c.Header("Content-Type", "video/webm")
		c.Header("Cache-Control", "public, max-age=3600")
		if _, err := io.Copy(c.Writer, rc); err != nil {
			log.Printf("[VIDEO] stream merged error: %v", err)
		}
		return
	}

	// Merged file doesn't exist — merge now.
	if err := h.MergeForAttempt(assignmentID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to process video")
		return
	}

	// Serve the newly merged file.
	rc, err = h.Storage.Get(c.Request.Context(), mergedKey)
	if err != nil {
		utils.NotFound(c, "merged video not found after processing")
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "video/webm")
	c.Header("Cache-Control", "public, max-age=3600")
	if _, err := io.Copy(c.Writer, rc); err != nil {
		log.Printf("[VIDEO] stream merged error: %v", err)
	}
}

// hasVideoProctoring checks whether an assignment has video_proctoring enabled.
func (h *VideoHandler) hasVideoProctoring(assignmentID int) (bool, error) {
	raw, err := h.AssignmentRepo.GetPolicy(assignmentID)
	if err != nil {
		return false, err
	}
	var p IntegrityPolicy
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return false, nil
		}
	}
	return p.VideoProctoring, nil
}

// MergeForAttempt merges video chunks into a single merged.webm. Safe to call from a goroutine.
func (h *VideoHandler) MergeForAttempt(assignmentID int) error {
	hasVP, err := h.hasVideoProctoring(assignmentID)
	if err != nil {
		return fmt.Errorf("check policy: %w", err)
	}
	if !hasVP {
		return nil
	}

	attemptID, _, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		return fmt.Errorf("no attempt found for assignment %d", assignmentID)
	}

	prefix := fmt.Sprintf("video/%d/", attemptID)
	keys, err := h.Storage.List(context.Background(), prefix)
	if err != nil {
		return fmt.Errorf("failed to list chunks: %w", err)
	}

	var chunkKeys []string
	for _, key := range keys {
		idx := strings.TrimPrefix(key, prefix)
		idx = strings.TrimSuffix(idx, ".webm")
		if idx != "" && idx != "merged" {
			chunkKeys = append(chunkKeys, key)
		}
	}

	if len(chunkKeys) == 0 {
		mergedKey := fmt.Sprintf("video/%d/merged.webm", attemptID)
		if _, err := h.Storage.Get(context.Background(), mergedKey); err == nil {
			return nil
		}
		return fmt.Errorf("no video chunks found")
	}

	sort.Slice(chunkKeys, func(i, j int) bool {
		a := strings.TrimSuffix(strings.TrimPrefix(chunkKeys[i], prefix), ".webm")
		b := strings.TrimSuffix(strings.TrimPrefix(chunkKeys[j], prefix), ".webm")
		ai, _ := strconv.Atoi(a)
		bi, _ := strconv.Atoi(b)
		return ai < bi
	})

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("video-merge-%d-*", assignmentID))
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for i, key := range chunkKeys {
		rc, err := h.Storage.Get(context.Background(), key)
		if err != nil {
			return fmt.Errorf("download chunk %s: %w", key, err)
		}
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%04d.webm", i))
		f, err := os.Create(chunkPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create chunk file: %w", err)
		}
		if _, err := io.Copy(f, rc); err != nil {
			f.Close()
			rc.Close()
			return fmt.Errorf("write chunk file: %w", err)
		}
		f.Close()
		rc.Close()
	}

	var listBuilder strings.Builder
	for i := range chunkKeys {
		listBuilder.WriteString(fmt.Sprintf("file '%04d.webm'\n", i))
	}
	listPath := filepath.Join(tmpDir, "filelist.txt")
	if err := os.WriteFile(listPath, []byte(listBuilder.String()), 0o644); err != nil {
		return fmt.Errorf("write concat list: %w", err)
	}

	mergedPath := filepath.Join(tmpDir, "merged.webm")
	cmd := exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", "filelist.txt", "-c", "copy", "merged.webm")
	cmd.Dir = tmpDir
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge failed: %s: %w", stderrBuf.String(), err)
	}

	// Delete chunks FIRST, then upload merged.
	if err := h.Storage.DeletePrefix(context.Background(), prefix); err != nil {
		log.Printf("[VIDEO] warning: failed to delete chunks for assignment %d: %v", assignmentID, err)
	}

	mergedFile, err := os.Open(mergedPath)
	if err != nil {
		return fmt.Errorf("open merged file: %w", err)
	}
	defer mergedFile.Close()

	mergedKey := fmt.Sprintf("video/%d/merged.webm", attemptID)
	if _, err := h.Storage.Put(context.Background(), mergedKey, mergedFile); err != nil {
		return fmt.Errorf("upload merged video: %w", err)
	}

	log.Printf("[VIDEO] merged %d chunks for assignment %d into merged.webm", len(chunkKeys), assignmentID)
	return nil
}

// DeleteVideo permanently deletes all video recordings for an assignment.
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
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

	hasVP, vpErr := h.hasVideoProctoring(assignmentID)
	if vpErr != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, vpErr, "failed to check policy")
		return
	}
	if !hasVP {
		c.JSON(http.StatusOK, gin.H{"message": "no video to delete"})
		return
	}

	attemptID, _, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		utils.NotFound(c, "no attempt found for this assignment")
		return
	}

	prefix := fmt.Sprintf("video/%d/", attemptID)
	if err := h.Storage.DeletePrefix(c.Request.Context(), prefix); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete video")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "video deleted permanently"})
}

func getRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
