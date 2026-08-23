package handlers

import (
	"net/http"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	ProfileRepo *repository.ProfileRepo
	UserRepo    *repository.UserRepo
}

func NewProfileHandler(profileRepo *repository.ProfileRepo, userRepo *repository.UserRepo) *ProfileHandler {
	return &ProfileHandler{ProfileRepo: profileRepo, UserRepo: userRepo}
}

// GET /auth/profile
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	profile, err := h.ProfileRepo.GetByUserID(userID)
	if err != nil {
		utils.NotFound(c, "profile not found")
		return
	}
	c.JSON(http.StatusOK, profile)
}

// PUT /auth/profile { display_name, phone }
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		Phone       string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := h.ProfileRepo.UpdateProfile(userID, req.DisplayName, req.Phone); err != nil {
		utils.InternalError(c, err, "failed to update profile")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

// PUT /auth/password { current_password, new_password }
func (h *ProfileHandler) UpdatePassword(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	currentHash, err := h.UserRepo.GetPasswordHash(userID)
	if err != nil {
		utils.NotFound(c, "user not found")
		return
	}
	if err := utils.CheckPassword(req.CurrentPassword, currentHash); err != nil {
		utils.BadRequest(c, "current password is incorrect")
		return
	}

	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.InternalError(c, err, "failed to hash password")
		return
	}
	if err := h.UserRepo.UpdatePassword(userID, hashed); err != nil {
		utils.InternalError(c, err, "failed to update password")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
