package auth

import (
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	AuthService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterAdminRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	OrgName  string `json:"org_name" binding:"required"`
}

func (h *AuthHandler) RegisterAdmin(c *gin.Context) {
	var req RegisterAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, err, "hashing failed")
		return
	}

	tenantID, userID, err := h.AuthService.RegisterAdmin(req.Email, hashed, req.OrgName)
	if err != nil {
		utils.BadRequest(c, "email already exists")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Organization registered successfully",
		"tenant_id": tenantID,
		"user_id":   userID,
		"role":      "admin",
	})
}

func (h *AuthHandler) UserLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	result, err := h.AuthService.UserLogin(req.Email, req.Password)
	if err != nil {
		utils.Unauthorized(c, "invalid credentials")
		return
	}

	token, err := utils.GenerateToken(result.UserID, result.Role, 0, int(result.TenantID))
	if err != nil {
		utils.InternalError(c, err, "token error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"role":      result.Role,
		"tenant_id": result.TenantID,
	})
}

type RegisterCoachRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) RegisterCoach(c *gin.Context) {
	var req RegisterCoachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	userID := c.GetInt("user_id")

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, err, "hashing failed")
		return
	}

	newUserID, coachID, err := h.AuthService.RegisterCoach(userID, req.Email, hashed, req.Name)
	if err != nil {
		utils.InternalError(c, err, "coach registration failed")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id":  newUserID,
		"coach_id": coachID,
		"email":    req.Email,
		"name":     req.Name,
		"role":     "coach",
	})
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID := c.GetInt("user_id")
	role := c.GetString("role")

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	if err := h.AuthService.UpdatePassword(userID, role, req.CurrentPassword, req.NewPassword); err != nil {
		utils.Unauthorized(c, "invalid credentials")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}


