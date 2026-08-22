package liveview

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type StudentWSHandler struct {
	Hub            *Hub
	StudentRepo    *repository.StudentRepo
	AssignmentRepo *repository.AssignmentRepo
}

func NewStudentWSHandler(hub *Hub, studentRepo *repository.StudentRepo, assignmentRepo *repository.AssignmentRepo) *StudentWSHandler {
	return &StudentWSHandler{
		Hub:            hub,
		StudentRepo:    studentRepo,
		AssignmentRepo: assignmentRepo,
	}
}

func (h *StudentWSHandler) StudentLiveStream(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		utils.Unauthorized(c, "missing token")
		return
	}

	claims, err := utils.ValidateToken(tokenStr)
	if err != nil {
		utils.Unauthorized(c, "invalid token")
		return
	}

	if claims.Role != "student" {
		utils.Unauthorized(c, "students only")
		return
	}

	studentID := claims.StudentID
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment_id")
		return
	}

	detail, err := h.AssignmentRepo.GetDetailForStudent(assignmentID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return
	}

	if detail.OwnerID != studentID {
		utils.Forbidden(c, "assignment does not belong to student")
		return
	}

	policy, err := h.AssignmentRepo.GetPolicy(assignmentID)
	if err != nil || len(policy) == 0 {
		utils.BadRequest(c, "no integrity policy")
		return
	}

	if !containsVideoProctoring(policy) {
		utils.BadRequest(c, "video proctoring not enabled")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[LIVEVIEW] WebSocket upgrade failed: %v", err)
		return
	}

	h.Hub.RegisterStudent(studentID, conn)
	defer h.Hub.UnregisterStudent(studentID)

	conn.SetReadLimit(512 * 1024)
	for {
		_, frame, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[LIVEVIEW] Student %d WS error: %v", studentID, err)
			}
			break
		}

		if len(frame) > 0 {
			h.Hub.RelayFrame(studentID, frame)
		}
	}
}

func containsVideoProctoring(policy []byte) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(policy, &raw); err != nil {
		s := string(policy)
		return strings.Contains(s, `"video_proctoring":true`) || strings.Contains(s, `"video_proctoring": true`)
	}

	val, ok := raw["video_proctoring"]
	if !ok {
		return false
	}

	var enabled bool
	if err := json.Unmarshal(val, &enabled); err == nil {
		return enabled
	}

	s := string(val)
	return strings.TrimSpace(s) == "true"
}
