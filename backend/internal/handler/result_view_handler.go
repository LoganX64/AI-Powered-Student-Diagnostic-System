package handlers

import (
	"net/http"

	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"github.com/gin-gonic/gin"
)

// GetResultByToken serves the emailed result link.
//
// Deliberately mounted OUTSIDE the authenticated student group: the token is the
// credential, and a student may click the link hours later without being signed
// in. Forcing a login would defeat the link.
//
// Returns 404 for every failure — unknown, expired, revoked, or released-then-
// withdrawn — so the endpoint cannot be used to discover which.
func (h *ResultHandler) GetResultByToken(c *gin.Context) {
	token := c.Param("token")
	if len(token) > 256 {
		// Bounded before any database work; a real token is 64 hex characters.
		utils.NotFound(c, "result not found")
		return
	}

	result, err := h.ViewService.GetByToken(token)
	if err != nil {
		if err == services.ErrResultNotFound {
			utils.NotFound(c, "result not found")
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to load result")
		return
	}

	// The token must not travel onward: any outbound request from this page
	// would otherwise carry it in the Referer header.
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Cache-Control", "no-store")

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// ListMyResults returns the signed-in student's released results.
//
// Mounted under the student auth group; the student id comes from the verified
// token in the cookie, never from the URL, so one student cannot ask for
// another's results.
func (h *ResultHandler) ListMyResults(c *gin.Context) {
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	results, err := h.ViewService.ListForStudent(studentID, c.GetInt("tenant_id"))
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to load results")
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}