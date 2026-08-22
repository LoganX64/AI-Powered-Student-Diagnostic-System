package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

// IntegrityPolicy mirrors the assignments.integrity_policy JSONB flags.
type IntegrityPolicy struct {
	ServerTiming     bool `json:"server_timing"`
	Autosave         bool `json:"autosave"`
	VideoProctoring  bool `json:"video_proctoring"`
	TabSwitchDetect  bool `json:"tab_switch_detect"`
}

func parsePolicy(raw []byte) IntegrityPolicy {
	var p IntegrityPolicy
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			// A corrupt policy must not silently disable proctoring. Fail
			// closed to the strictest policy and log so it can be repaired.
			log.Printf("[POLICY] invalid integrity_policy JSON, defaulting to strict: %v", err)
			return IntegrityPolicy{ServerTiming: true, Autosave: true, VideoProctoring: true, TabSwitchDetect: true}
		}
	}
	return p
}

func (h *StudentHandler) loadPolicyAndOwnership(c *gin.Context, assignmentID, studentID int) (IntegrityPolicy, int, bool) {
	detail, err := h.AssignmentRepo.GetDetailForStudent(assignmentID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return IntegrityPolicy{}, 0, false
	}
	if detail.OwnerID != studentID {
		utils.Forbidden(c, "assignment does not belong to student")
		return IntegrityPolicy{}, 0, false
	}
	raw, err := h.AssignmentRepo.GetPolicy(assignmentID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return IntegrityPolicy{}, 0, false
	}
	return parsePolicy(raw), detail.Duration, true
}

func (h *StudentHandler) graceSeconds() int {
	if h.Cfg != nil && h.Cfg.SubmitGraceSeconds > 0 {
		return h.Cfg.SubmitGraceSeconds
	}
	return 30
}

// StartExam records the authoritative start time (server_timing tier only).
func (h *StudentHandler) StartExam(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	policy, duration, ok := h.loadPolicyAndOwnership(c, assignmentID, studentID)
	if !ok {
		return
	}
	if !policy.ServerTiming {
		utils.Conflict(c, "this exam uses client-side timing; /start is not required")
		return
	}

	var attemptID int
	var startedAt time.Time
	if existingID, t, err := h.AttemptRepo.GetInProgressAttempt(assignmentID); err == nil {
		attemptID, startedAt = existingID, t
	} else {
		id, t, err := h.AttemptRepo.CreateInProgressAttempt(assignmentID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to start exam")
			return
		}
		attemptID, startedAt = id, t
	}

	deadline := startedAt.Add(time.Duration(duration) * time.Minute)
	c.JSON(http.StatusOK, gin.H{
		"attempt_id": attemptID,
		"deadline":   deadline.Format(time.RFC3339),
		"server_now": time.Now().Format(time.RFC3339),
	})
}

// Autosave persists answers to the server (autosave tier).
func (h *StudentHandler) Autosave(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	policy, _, ok := h.loadPolicyAndOwnership(c, assignmentID, studentID)
	if !ok {
		return
	}
	if !policy.Autosave {
		utils.Conflict(c, "autosave is not enabled for this exam")
		return
	}

	var req struct {
		Answers []services.AnswerInput `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	attemptID, _, err := h.AttemptRepo.GetInProgressAttempt(assignmentID)
	if err != nil {
		id, _, err := h.AttemptRepo.CreateInProgressAttempt(assignmentID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to start attempt")
			return
		}
		attemptID = id
	}

	// Scale mode: buffer writes in Redis and flush in batches (surge safety).
	if h.AutosaveBuffer != nil {
		if err := h.AutosaveBuffer.Push(attemptID, req.Answers); err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to buffer answers")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"saved":    len(req.Answers),
			"buffered": len(req.Answers),
		})
		return
	}

	saved := 0
	for _, ans := range req.Answers {
		answerSeen := ans.SelectedAnswer != ""
		if ans.Seen != nil {
			answerSeen = *ans.Seen
		}
		if !answerSeen {
			ans.TimeSpent = 0
			ans.SelectedAnswer = ""
			ans.MarkedForReview = false
			ans.Revisited = false
			ans.ChangedAnswer = false
			ans.WasInitiallyWrong = false
		}
		if err := h.AttemptRepo.UpsertAnswer(
			attemptID, ans.QuestionID, ans.SelectedAnswer, false, ans.TimeSpent,
			ans.MarkedForReview, ans.Revisited, ans.ChangedAnswer, ans.WasInitiallyWrong, answerSeen,
		); err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to autosave")
			return
		}
		saved++
	}

	c.JSON(http.StatusOK, gin.H{"saved": saved})
}

// GetState returns remaining time + saved answers for resume (autosave/server_timing).
func (h *StudentHandler) GetState(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	policy, duration, ok := h.loadPolicyAndOwnership(c, assignmentID, studentID)
	if !ok {
		return
	}
	if !policy.ServerTiming && !policy.Autosave {
		utils.Conflict(c, "resume is not available for this exam")
		return
	}

	attemptID, startedAt, err := h.AttemptRepo.GetInProgressAttempt(assignmentID)
	if err != nil {
		utils.NotFound(c, "exam not started")
		return
	}

	deadline := startedAt.Add(time.Duration(duration) * time.Minute)
	remaining := int(time.Until(deadline).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	answers, err := h.AttemptRepo.GetSavedAnswers(attemptID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to load state")
		return
	}
	if answers == nil {
		answers = []repository.SavedAnswer{}
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt_id":        attemptID,
		"deadline":          deadline.Format(time.RFC3339),
		"remaining_seconds": remaining,
		"answers":           answers,
	})
}

// SubmitExam finalizes the attempt. For server_timing it validates the deadline;
// otherwise it falls back to the existing client-time submit path. No scoring here.
func (h *StudentHandler) SubmitExam(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	policy, duration, ok := h.loadPolicyAndOwnership(c, assignmentID, studentID)
	if !ok {
		return
	}

	var req struct {
		Answers []services.AnswerInput `json:"answers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	if policy.ServerTiming {
		if h.Cfg.RedisEnabled {
			// Scale mode: validate synchronously, then enqueue finalization so
			// the DB write is decoupled from the request (accept-fast pattern).
			attemptID, err := h.AttemptService.ValidateTimedSubmit(assignmentID, studentID, h.graceSeconds())
			if err != nil {
				var svcErr *services.SubmitAnswersError
				if errors.As(err, &svcErr) {
					c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
					return
				}
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to submit")
				return
			}
			// Compute the same summary fields the synchronous path returns so
			// the response contract is identical in scale mode.
			var totalTimeSpent float64
			for _, a := range req.Answers {
				totalTimeSpent += a.TimeSpent
			}
			if err := h.Queue.EnqueueFinalize(queue.FinalizePayload{
				AssignmentID: assignmentID,
				AttemptID:    attemptID,
				StudentID:    studentID,
				Answers:      toQueueAnswers(req.Answers),
			}); err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to enqueue finalization")
				return
			}
			c.JSON(http.StatusAccepted, gin.H{
				"attempt_id":       attemptID,
				"status":           "queued",
				"total_time_spent": helper.Round2V2(totalTimeSpent),
				"test_duration":    float64(duration),
			})
			if policy.VideoProctoring && h.VideoHandler != nil {
				go h.VideoHandler.MergeForAttempt(assignmentID)
			}
			return
		}

		res, err := h.AttemptService.SubmitTimed(assignmentID, studentID, h.graceSeconds(), req.Answers)
		if err != nil {
			var svcErr *services.SubmitAnswersError
			if errors.As(err, &svcErr) {
				c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
				return
			}
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to submit")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"attempt_id":       res.AttemptID,
			"total_time_spent": res.TotalTimeSpent,
			"test_duration":    res.TestDuration,
		})
		if policy.VideoProctoring && h.VideoHandler != nil {
			go h.VideoHandler.MergeForAttempt(assignmentID)
		}
		return
	}

	// Simple tier: delegate to the existing finalize path.
	res, err := h.AttemptService.SubmitAnswers(assignmentID, studentID, req.Answers)
	if err != nil {
		var svcErr *services.SubmitAnswersError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to submit")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"attempt_id":       res.AttemptID,
		"total_time_spent": res.TotalTimeSpent,
		"test_duration":    res.TestDuration,
	})
	if policy.VideoProctoring && h.VideoHandler != nil {
		go h.VideoHandler.MergeForAttempt(assignmentID)
	}
}

// VideoChunk accepts an indexed video blob (video_proctoring tier, record-only).
func (h *StudentHandler) VideoChunk(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	policy, _, ok := h.loadPolicyAndOwnership(c, assignmentID, studentID)
	if !ok {
		return
	}
	if !policy.VideoProctoring {
		utils.Conflict(c, "video proctoring is not enabled for this exam")
		return
	}

	attemptID, _, err := h.AttemptRepo.GetInProgressAttempt(assignmentID)
	if err != nil {
		utils.Conflict(c, "start the exam before uploading video")
		return
	}

	index := c.PostForm("index")
	chunk, err := c.FormFile("chunk")
	if err != nil {
		utils.BadRequest(c, "missing chunk file")
		return
	}

	key := fmt.Sprintf("video/%d/%s.webm", attemptID, index)
	src, err := chunk.Open()
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to read chunk")
		return
	}
	defer src.Close()

	storedURL, err := h.Storage.Put(c.Request.Context(), key, src)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to store chunk")
		return
	}

	c.JSON(http.StatusOK, gin.H{"received_index": index, "url": storedURL})
}

// ServerTime returns the authoritative server clock for client skew calibration.
func (h *StudentHandler) ServerTime(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"server_time": time.Now().Format(time.RFC3339)})
}

// toQueueAnswers converts handler answer inputs into the queue-local shape so
// the services package is not imported by the queue package (avoids a cycle).
func toQueueAnswers(in []services.AnswerInput) []queue.AnswerInput {
	out := make([]queue.AnswerInput, len(in))
	for i, a := range in {
		out[i] = queue.AnswerInput{
			QuestionID:        a.QuestionID,
			SelectedAnswer:    a.SelectedAnswer,
			TimeSpent:         a.TimeSpent,
			Seen:              a.Seen,
			MarkedForReview:   a.MarkedForReview,
			Revisited:         a.Revisited,
			ChangedAnswer:     a.ChangedAnswer,
			WasInitiallyWrong: a.WasInitiallyWrong,
		}
	}
	return out
}
