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

// scopeFor maps the caller's role to the rows they may read. Every tenant user,
// including the admin, sees only rows addressed to them; the old tenant-wide
// admin view let one admin read another admin's copy and is gone. Super admins
// have no tenant and read platform rows only.
func scopeFor(role string) repository.NotificationScope {
	if role == "super_admin" {
		return repository.ScopePlatform
	}
	return repository.ScopeOwn
}

// scopeTenantID returns the tenant id for tenant-scoped calls and nil for
// platform scope, which has no tenant predicate.
func scopeTenantID(c *gin.Context, scope repository.NotificationScope) *int {
	if scope == repository.ScopePlatform {
		return nil
	}
	tenantID := c.GetInt("tenant_id")
	return &tenantID
}

// parseTriBool maps a query-string flag to the repository's tri-state filters:
// nil means "any", true means "must be set", false means "must be NULL". An
// absent or unrecognised value is "any" rather than an error, matching the
// existing priority filter's forgiving behavior.
func parseTriBool(raw string) *bool {
	switch raw {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}

// normalizePriority accepts only the severities the notifications table defines.
// An unrecognised value is dropped rather than passed through, so a typo or a
// stale client asking for a severity that no longer exists returns the unfiltered
// list instead of a permanently empty page.
func normalizePriority(p string) string {
	switch p {
	case "info", "warning", "alert":
		return p
	}
	return ""
}

// GET /admin/notifications?event_type=&priority=&read=&unread=&seen=&limit=&offset=
// GET /coach/notifications (same filters)
// GET /super-admin/notifications (same filters, platform scope)
//
// read=true|false and seen=true|false select rows with that timestamp set or
// still NULL; unread=true is kept as sugar for read=false. event_type and
// priority narrow independently. The response carries viewer_user_id so the
// client can tell which rows it may act on: every scope is owner-scoped for
// mutation, so without this the UI would offer actions that 404.
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))

	filter := repository.NotificationFilter{
		TenantID:  scopeTenantID(c, scope),
		UserID:    userID,
		Scope:     scope,
		EventType: c.Query("event_type"),
		Priority:  normalizePriority(c.Query("priority")),
		Read:      parseTriBool(c.Query("read")),
		Seen:      parseTriBool(c.Query("seen")),
		Limit:     limit,
		Offset:    offset,
	}
	if c.Query("unread") == "true" && filter.Read == nil {
		unread := false
		filter.Read = &unread
	}

	notifications, total, err := h.NotificationRepo.ListFiltered(filter)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch notifications")
		return
	}
	if notifications == nil {
		notifications = []repository.NotificationRow{}
	}
	c.JSON(http.StatusOK, gin.H{
		"total": total, "limit": limit, "offset": offset,
		"data": notifications, "viewer_user_id": userID,
	})
}

// GET /admin/notifications/unread-count
//
// unseen_count is the badge number: it counts rows the client has not opened,
// whether or not they were marked read. unread_count is the Unread tab and only
// drops when the row is explicitly marked read.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	counts, err := h.NotificationRepo.Counts(scopeTenantID(c, scope), userID, scope)
	if err != nil {
		utils.InternalError(c, err, "failed to count unread")
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread_count": counts.Unread, "unseen_count": counts.Unseen})
}

// PUT /admin/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid notification id")
		return
	}
	// Scoped to the caller, so another user's notification is a 404 here rather
	// than a silent 200.
	found, err := h.NotificationRepo.MarkReadScoped(id, scopeTenantID(c, scope), userID, scope)
	if err != nil {
		utils.InternalError(c, err, "failed to mark as read")
		return
	}
	if !found {
		utils.NotFound(c, "notification not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

// PUT /admin/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	if err := h.NotificationRepo.MarkAllReadScoped(scopeTenantID(c, scope), userID, scope); err != nil {
		utils.InternalError(c, err, "failed to mark all as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
}

// PUT /admin/notifications/:id/seen
//
// Marking seen clears the badge without touching read_at, so a row can leave the
// badge while staying in the Unread tab.
func (h *NotificationHandler) MarkSeen(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid notification id")
		return
	}
	found, err := h.NotificationRepo.MarkSeenScoped(id, scopeTenantID(c, scope), userID, scope)
	if err != nil {
		utils.InternalError(c, err, "failed to mark as seen")
		return
	}
	if !found {
		utils.NotFound(c, "notification not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marked as seen"})
}

// PUT /admin/notifications/seen-all
func (h *NotificationHandler) MarkAllSeen(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	if err := h.NotificationRepo.MarkAllSeenScoped(scopeTenantID(c, scope), userID, scope); err != nil {
		utils.InternalError(c, err, "failed to mark all as seen")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "all marked as seen"})
}

// DELETE /admin/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID := c.GetInt("user_id")
	scope := scopeFor(c.GetString("role"))
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid notification id")
		return
	}
	found, err := h.NotificationRepo.DeleteScoped(id, scopeTenantID(c, scope), userID, scope)
	if err != nil {
		utils.InternalError(c, err, "failed to delete notification")
		return
	}
	if !found {
		utils.NotFound(c, "notification not found")
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
//
// Every key must be a known event type. Retired types (system_alert,
// storage_warning) and typos are rejected instead of being stored as dead
// preferences that can never fire.
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req struct {
		Preferences map[string]bool `json:"preferences" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	for eventType := range req.Preferences {
		if !services.IsValidEventType(eventType) {
			utils.BadRequest(c, "unknown notification event type: "+eventType)
			return
		}
	}
	if err := h.NotificationRepo.UpdatePreferences(userID, req.Preferences); err != nil {
		utils.InternalError(c, err, "failed to update preferences")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "preferences updated"})
}
