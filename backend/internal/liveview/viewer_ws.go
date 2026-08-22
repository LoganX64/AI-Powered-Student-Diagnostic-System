package liveview

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ViewerWSHandler struct {
	Hub            *Hub
	StudentRepo    *repository.StudentRepo
	AssignmentRepo *repository.AssignmentRepo
	CoachRepo      *repository.CoachRepo
}

func NewViewerWSHandler(
	hub *Hub,
	studentRepo *repository.StudentRepo,
	assignmentRepo *repository.AssignmentRepo,
	coachRepo *repository.CoachRepo,
) *ViewerWSHandler {
	return &ViewerWSHandler{
		Hub:            hub,
		StudentRepo:    studentRepo,
		AssignmentRepo: assignmentRepo,
		CoachRepo:      coachRepo,
	}
}

func (h *ViewerWSHandler) ViewerLiveStream(c *gin.Context) {
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

	if claims.Role != "admin" && claims.Role != "coach" {
		utils.Unauthorized(c, "admin or coach role required")
		return
	}

	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student_id")
		return
	}

	studentCoachID, studentTenantID, err := h.StudentRepo.GetCoachIDAndTenantID(studentID)
	if err != nil {
		utils.NotFound(c, "student not found")
		return
	}

	if claims.TenantID != studentTenantID {
		utils.Forbidden(c, "student not in your organization")
		return
	}

	if claims.Role == "coach" {
		viewerCoachID, err := h.CoachRepo.GetIDFromUser(claims.UserID)
		if err != nil {
			utils.InternalError(c, err, "coach profile not found")
			return
		}
		if viewerCoachID != studentCoachID {
			utils.Forbidden(c, "student not assigned to you")
			return
		}
	}

	if !h.Hub.IsLive(studentID) {
		utils.BadRequest(c, "student is not currently live")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[LIVEVIEW] Viewer WS upgrade failed: %v", err)
		return
	}

	if err := h.Hub.AddViewer(studentID, conn); err != nil {
		conn.Close()
		return
	}
	defer h.Hub.RemoveViewer(studentID, conn)

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go h.viewerReadLoop(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[LIVEVIEW] Viewer WS error: %v", err)
			}
			break
		}
	}
}

func (h *ViewerWSHandler) viewerReadLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

func (h *ViewerWSHandler) LiveStatus(c *gin.Context) {
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

	if claims.Role != "admin" && claims.Role != "coach" {
		utils.Unauthorized(c, "admin or coach role required")
		return
	}

	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student_id")
		return
	}

	studentCoachID, studentTenantID, err := h.StudentRepo.GetCoachIDAndTenantID(studentID)
	if err != nil {
		utils.NotFound(c, "student not found")
		return
	}

	if claims.TenantID != studentTenantID {
		utils.Forbidden(c, "student not in your organization")
		return
	}

	if claims.Role == "coach" {
		viewerCoachID, err := h.CoachRepo.GetIDFromUser(claims.UserID)
		if err != nil {
			utils.InternalError(c, err, "coach profile not found")
			return
		}
		if viewerCoachID != studentCoachID {
			utils.Forbidden(c, "student not assigned to you")
			return
		}
	}

	live := h.Hub.IsLive(studentID)
	c.JSON(http.StatusOK, gin.H{"live": live})
}
