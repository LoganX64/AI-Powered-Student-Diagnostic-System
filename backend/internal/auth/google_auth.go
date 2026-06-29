package auth

import (
	"ai-student-diagnostic/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GoogleRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req GoogleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	result, err := h.AuthService.GoogleLogin(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google token"})
		return
	}

	token, err := utils.GenerateToken(result.UserID, result.Role, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"role":      result.Role,
		"tenant_id": result.TenantID,
	})
}
