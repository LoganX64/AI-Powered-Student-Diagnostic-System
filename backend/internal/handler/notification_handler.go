package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	NotificationService *services.NotificationService
	NotificationRepo    *repository.NotificationRepo
}

func NewNotificationHandler(
	notifService *services.NotificationService,
	notifRepo *repository.NotificationRepo,
) *NotificationHandler {
	return &NotificationHandler{
		NotificationService: notifService,
		NotificationRepo:    notifRepo,
	}
}

// GET /admin/notifications?event_type=&unread=&limit=&offset=
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	eventType := c.Query("event_type")
	unreadOnly := c.Query("unread") == "true"
	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))

	notifications, total, err := h.NotificationRepo.List(tenantID, &userID, eventType, unreadOnly, limit, offset)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch notifications")
		return
	}
	if notifications == nil {
		notifications = []repository.NotificationRow{}
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": notifications})
}

// GET /admin/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	count, err := h.NotificationRepo.UnreadCount(tenantID, &userID)
	if err != nil {
		utils.InternalError(c, err, "failed to count unread")
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// PUT /admin/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid notification id")
		return
	}
	if err := h.NotificationRepo.MarkRead(id, tenantID); err != nil {
		utils.InternalError(c, err, "failed to mark as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

// PUT /admin/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if err := h.NotificationRepo.MarkAllRead(tenantID, &userID); err != nil {
		utils.InternalError(c, err, "failed to mark all as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
}

// DELETE /admin/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid notification id")
		return
	}
	if err := h.NotificationRepo.Delete(id, tenantID); err != nil {
		utils.InternalError(c, err, "failed to delete notification")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
}

// GET /admin/notifications/preferences
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := c.GetInt("user_id")
	prefs, err := h.NotificationRepo.GetPreferences(userID)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch preferences")
		return
	}
	if prefs == nil {
		prefs = []repository.NotificationPrefRow{}
	}
	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

// PUT /admin/notifications/preferences { preferences: { event_type: bool } }
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req struct {
		Preferences map[string]bool `json:"preferences" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := h.NotificationRepo.UpdatePreferences(userID, req.Preferences); err != nil {
		utils.InternalError(c, err, "failed to update preferences")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}
