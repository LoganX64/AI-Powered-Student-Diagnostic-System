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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var inMemoryVideoMergeStatus sync.Map // map[int]string

type VideoHandler struct {
	AttemptRepo    *repository.AttemptRepo
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	CoachRepo      *repository.CoachRepo
	Storage        storage.Storage
	RDB            *redis.Client
}

func NewVideoHandler(
	attemptRepo *repository.AttemptRepo,
	assignmentRepo *repository.AssignmentRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	storageBackend storage.Storage,
	rdb *redis.Client,
) *VideoHandler {
	return &VideoHandler{
		AttemptRepo:    attemptRepo,
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		CoachRepo:      coachRepo,
		Storage:        storageBackend,
		RDB:            rdb,
	}
}

func (h *VideoHandler) getMergeStatus(attemptID int) string {
	if h.RDB != nil {
		val, err := h.RDB.Get(context.Background(), fmt.Sprintf("video:merge:status:%d", attemptID)).Result()
		if err == nil {
			return val
		}
	}
	if val, ok := inMemoryVideoMergeStatus.Load(attemptID); ok {
		return val.(string)
	}
	return ""
}

func (h *VideoHandler) setMergeStatus(attemptID int, status string) {
	if h.RDB != nil {
		ttl := 15 * time.Minute
		if status == "ready" {
			ttl = 24 * time.Hour
		}
		_ = h.RDB.Set(context.Background(), fmt.Sprintf("video:merge:status:%d", attemptID), status, ttl).Err()
	}
	inMemoryVideoMergeStatus.Store(attemptID, status)
}

func (h *VideoHandler) StartMergeAsync(assignmentID, attemptID int) {
	if h.getMergeStatus(attemptID) == "processing" {
		return
	}
	h.setMergeStatus(attemptID, "processing")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[VIDEO] panic in async merge for assignment %d: %v", assignmentID, r)
				h.setMergeStatus(attemptID, "failed")
			}
		}()

		if err := h.MergeForAttempt(assignmentID); err != nil {
			log.Printf("[VIDEO] async merge failed for assignment %d: %v", assignmentID, err)
			h.setMergeStatus(attemptID, "failed")
			return
		}

		h.setMergeStatus(attemptID, "ready")
	}()
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

// GenerateVideoToken issues a short-lived JWT for video playback.
func (h *VideoHandler) GenerateVideoToken(c *gin.Context) {
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

	tenantID := c.GetInt("tenant_id")
	token, err := utils.GenerateVideoToken(assignmentID, tenantID, role)
	if err != nil {
		utils.InternalError(c, err, "failed to generate video token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"expires_in":   300,
		"assignment_id": assignmentID,
	})
}

// VideoStatus returns the status of an assignment's video merge.
// If not yet merged and chunks exist, it starts a background merge and returns 202.
func (h *VideoHandler) VideoStatus(c *gin.Context) {
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

	// 1. Check if already merged
	if rc, err := h.Storage.Get(c.Request.Context(), mergedKey); err == nil {
		rc.Close()
		h.setMergeStatus(attemptID, "ready")
		c.JSON(http.StatusOK, gin.H{
			"assignment_id": assignmentID,
			"attempt_id":    attemptID,
			"status":        "ready",
			"has_merged":    true,
		})
		return
	}

	// 2. Check if chunks exist
	prefix := fmt.Sprintf("video/%d/", attemptID)
	keys, err := h.Storage.List(c.Request.Context(), prefix)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to check video chunks")
		return
	}

	var chunkCount int
	for _, key := range keys {
		idx := strings.TrimPrefix(key, prefix)
		idx = strings.TrimSuffix(idx, ".webm")
		if idx != "" && idx != "merged" {
			chunkCount++
		}
	}

	if chunkCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"assignment_id": assignmentID,
			"attempt_id":    attemptID,
			"status":        "none",
			"has_merged":    false,
			"chunks_count":  0,
		})
		return
	}

	// 3. Check current processing status
	status := h.getMergeStatus(attemptID)
	retry := c.Query("retry") == "true"

	if status == "processing" {
		c.JSON(http.StatusAccepted, gin.H{
			"assignment_id": assignmentID,
			"attempt_id":    attemptID,
			"status":        "processing",
			"has_merged":    false,
			"chunks_count":  chunkCount,
			"message":       "video merge in progress",
		})
		return
	}

	if status == "failed" && !retry {
		c.JSON(http.StatusOK, gin.H{
			"assignment_id": assignmentID,
			"attempt_id":    attemptID,
			"status":        "failed",
			"has_merged":    false,
			"chunks_count":  chunkCount,
			"message":       "video merge failed; retry available",
		})
		return
	}

	// 4. Trigger merge in background
	h.StartMergeAsync(assignmentID, attemptID)

	c.JSON(http.StatusAccepted, gin.H{
		"assignment_id": assignmentID,
		"attempt_id":    attemptID,
		"status":        "processing",
		"has_merged":    false,
		"chunks_count":  chunkCount,
		"message":       "video merge started in background",
	})
}

// StreamMergedVideo serves the merged video. If not merged, it triggers an async
// merge and returns 202 Accepted instead of blocking the Gin handler goroutine.
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

	// Merged file doesn't exist — kick off async merge without blocking HTTP handler.
	if h.getMergeStatus(attemptID) != "processing" {
		h.StartMergeAsync(assignmentID, attemptID)
	}

	c.Header("Retry-After", "5")
	c.JSON(http.StatusAccepted, gin.H{
		"assignment_id": assignmentID,
		"attempt_id":    attemptID,
		"status":        "processing",
		"message":       "video merge in progress, poll /assignments/:id/video-status",
	})
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
// It verifies the merged file before deleting any original chunks. If already merged, it does nothing.
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

	mergedKey := fmt.Sprintf("video/%d/merged.webm", attemptID)

	// Do not merge video which is already merged.
	if rc, err := h.Storage.Get(context.Background(), mergedKey); err == nil {
		rc.Close()
		h.setMergeStatus(attemptID, "ready")
		return nil
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

	// Verification 1: Confirm merged file on disk exists and is non-empty
	fi, err := os.Stat(mergedPath)
	if err != nil {
		return fmt.Errorf("stat merged video failed: %w", err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("merged video file on disk is empty (0 bytes)")
	}

	// Upload to storage
	mergedFile, err := os.Open(mergedPath)
	if err != nil {
		return fmt.Errorf("open merged file: %w", err)
	}
	defer mergedFile.Close()

	if _, err := h.Storage.Put(context.Background(), mergedKey, mergedFile); err != nil {
		return fmt.Errorf("upload merged video: %w", err)
	}

	// Verification 2: Read-back confirmation from storage before deleting chunks
	verifyRC, err := h.Storage.Get(context.Background(), mergedKey)
	if err != nil {
		return fmt.Errorf("verification failed: could not retrieve %s after upload: %w", mergedKey, err)
	}
	var testBuf [1]byte
	n, readErr := verifyRC.Read(testBuf[:])
	verifyRC.Close()
	if readErr != nil && readErr != io.EOF {
		return fmt.Errorf("verification failed: cannot read uploaded video %s: %w", mergedKey, readErr)
	}
	if n == 0 && readErr == io.EOF {
		return fmt.Errorf("verification failed: uploaded video %s is 0 bytes", mergedKey)
	}

	// Safe Deletion: Chunks are deleted ONLY after the merged file has passed all checks.
	for _, key := range chunkKeys {
		if err := h.Storage.Delete(context.Background(), key); err != nil {
			log.Printf("[VIDEO] warning: failed to delete chunk %s for assignment %d: %v", key, assignmentID, err)
		}
	}

	h.setMergeStatus(attemptID, "ready")
	log.Printf("[VIDEO] verified merged video and deleted %d chunks for assignment %d into merged.webm", len(chunkKeys), assignmentID)
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
