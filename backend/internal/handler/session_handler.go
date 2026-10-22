package handlers

import (
	"net/http"
	"slices"
	"time"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

// SessionHandler resolves the browser's role cookies in one request. The SPA
// uses it instead of probing role-gated endpoints, which would fan out into
// several requests per public page load and couple auth checks to feature APIs.
type SessionHandler struct {
	UserRepo    *repository.UserRepo
	StudentRepo *repository.StudentRepo
}

func NewSessionHandler(userRepo *repository.UserRepo, studentRepo *repository.StudentRepo) *SessionHandler {
	return &SessionHandler{UserRepo: userRepo, StudentRepo: studentRepo}
}

type sessionEntry struct {
	role     string
	issuedAt time.Time
}

// GetSession validates every role cookie independently and returns all roles
// that are still valid. Returning the full set preserves the app's multi-role
// sessions: a browser signed in as both admin and coach can use both
// dashboards without discarding one session.
func (h *SessionHandler) GetSession(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	entries := make([]sessionEntry, 0, len(utils.AuthRoles))
	for _, role := range utils.AuthRoles {
		raw, err := c.Cookie(utils.AuthCookieName(role))
		if err != nil || raw == "" {
			continue
		}

		claims, err := utils.ValidateToken(raw)
		if err != nil || claims.Role != role {
			continue
		}

		if role == "student" {
			if claims.StudentID == 0 {
				continue
			}
			exists, err := h.StudentRepo.ExistsByID(claims.StudentID)
			if err != nil {
				utils.InternalError(c, err, "session check failed")
				return
			}
			if !exists {
				continue
			}
		} else {
			if claims.UserID == 0 {
				continue
			}
			exists, err := h.UserRepo.ExistsByID(claims.UserID)
			if err != nil {
				utils.InternalError(c, err, "session check failed")
				return
			}
			if !exists {
				continue
			}
		}

		var issuedAt time.Time
		if claims.IssuedAt != nil {
			issuedAt = claims.IssuedAt.Time
		}
		entries = append(entries, sessionEntry{role: role, issuedAt: issuedAt})
	}

	if len(entries) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"role":          nil,
			"roles":         []string{},
		})
		return
	}

	slices.SortStableFunc(entries, func(a, b sessionEntry) int {
		return b.issuedAt.Compare(a.issuedAt)
	})

	roles := make([]string, len(entries))
	for i, entry := range entries {
		roles[i] = entry.role
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"role":          roles[0],
		"roles":         roles,
	})
}
