package handlers

import (
	"net/http"
	"strconv"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

type SuperAdminHandler struct {
	TenantRepo  *repository.TenantRepo
	ProfileRepo *repository.ProfileRepo
	AuthService *services.AuthService
}

func NewSuperAdminHandler(
	tenantRepo *repository.TenantRepo,
	profileRepo *repository.ProfileRepo,
	authService *services.AuthService,
) *SuperAdminHandler {
	return &SuperAdminHandler{
		TenantRepo:  tenantRepo,
		ProfileRepo: profileRepo,
		AuthService: authService,
	}
}

// GET /super-admin/tenants?search=&limit=&offset=
func (h *SuperAdminHandler) ListTenants(c *gin.Context) {
	search := c.Query("search")
	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))

	tenants, total, err := h.TenantRepo.List(search, limit, offset)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenants")
		return
	}
	if tenants == nil {
		tenants = []repository.TenantRow{}
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tenants})
}

// POST /super-admin/tenants { name, admin_email, admin_password, admin_name }
func (h *SuperAdminHandler) CreateTenant(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		AdminEmail    string `json:"admin_email" binding:"required"`
		AdminPassword string `json:"admin_password" binding:"required"`
		AdminName     string `json:"admin_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	if err := utils.ValidatePassword(req.AdminPassword); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	hashed, err := utils.HashPassword(req.AdminPassword)
	if err != nil {
		utils.InternalError(c, err, "failed to hash password")
		return
	}

	tenantID, userID, err := h.AuthService.RegisterAdmin(req.AdminEmail, hashed, req.Name)
	if err != nil {
		utils.InternalError(c, err, "failed to create tenant")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"tenant_id": tenantID, "user_id": userID})
}

// GET /super-admin/tenants/:id
func (h *SuperAdminHandler) GetTenant(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}

	tenant, err := h.TenantRepo.GetByID(tenantID)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant")
		return
	}
	if tenant == nil {
		utils.NotFound(c, "tenant not found")
		return
	}
	c.JSON(http.StatusOK, tenant)
}

// PUT /super-admin/tenants/:id { name }
func (h *SuperAdminHandler) UpdateTenant(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
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
		utils.InternalError(c, err, "failed to update tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant updated"})
}

// PUT /super-admin/tenants/:id/suspend
func (h *SuperAdminHandler) SuspendTenant(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}
	if err := h.TenantRepo.Suspend(tenantID); err != nil {
		utils.InternalError(c, err, "failed to suspend tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant suspended"})
}

// PUT /super-admin/tenants/:id/reactivate
func (h *SuperAdminHandler) ReactivateTenant(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}
	if err := h.TenantRepo.Reactivate(tenantID); err != nil {
		utils.InternalError(c, err, "failed to reactivate tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant reactivated"})
}

// GET /super-admin/tenants/:id/admins
func (h *SuperAdminHandler) ListTenantAdmins(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}
	admins, err := h.TenantRepo.GetAdmins(tenantID)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch admins")
		return
	}
	if admins == nil {
		admins = []repository.UserRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": admins})
}

// POST /super-admin/tenants/:id/admins { email, password, name }
func (h *SuperAdminHandler) CreateTenantAdmin(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalError(c, err, "failed to hash password")
		return
	}

	userID, err := h.AuthService.CreateAdminForTenant(tenantID, req.Email, hashed, req.Name)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user_id": userID})
}

// GET /super-admin/stats
func (h *SuperAdminHandler) GetGlobalStats(c *gin.Context) {
	stats, err := h.TenantRepo.GetGlobalStats()
	if err != nil {
		utils.InternalError(c, err, "failed to fetch stats")
		return
	}
	c.JSON(http.StatusOK, stats)
}
