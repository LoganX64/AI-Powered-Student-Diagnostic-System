package handlers

import (
	"net/http"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

type TenantSettingsHandler struct {
	TenantRepo *repository.TenantRepo
}

func NewTenantSettingsHandler(tenantRepo *repository.TenantRepo) *TenantSettingsHandler {
	return &TenantSettingsHandler{TenantRepo: tenantRepo}
}

// GET /admin/tenant/settings
func (h *TenantSettingsHandler) GetSettings(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	settings, err := h.TenantRepo.GetSettings(tenantID)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// PUT /admin/tenant/settings { key, value }
func (h *TenantSettingsHandler) UpdateSettings(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		Key   string      `json:"key" binding:"required"`
		Value interface{} `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := h.TenantRepo.UpsertSetting(tenantID, req.Key, req.Value); err != nil {
		utils.InternalError(c, err, "failed to update tenant settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

// PUT /admin/tenant { name }
func (h *TenantSettingsHandler) UpdateTenantName(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		utils.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := h.TenantRepo.Update(tenantID, req.Name); err != nil {
		utils.InternalError(c, err, "failed to update tenant name")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant name updated"})
}
