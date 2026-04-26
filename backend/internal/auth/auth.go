package auth

import (
	"ai-student-diagnostic/backend/utils"
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

// login (super_admin, admin, coach)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Start transaction
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}

	// 1. Create Tenant
	var tenantID int
	err = tx.QueryRow("INSERT INTO tenants (name) VALUES ($1) RETURNING id", req.OrgName).Scan(&tenantID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	// 2. Hash Password
	hashed, _ := utils.HashPassword(req.Password)

	// 3. Create Admin User
	var userID int
	err = tx.QueryRow(`
		INSERT INTO users (tenant_id, email, password, role)
		VALUES ($1, $2, $3, 'admin')
		RETURNING id
	`, tenantID, req.Email, hashed).Scan(&userID)

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Admin and Organization registered successfully",
		"tenant_id": tenantID,
		"user_id":   userID,
	})
}

func (h *AuthHandler) UserLogin(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var userID int
	var hashedPassword string
	var role string
	var tenantID sql.NullInt32

	err := h.DB.QueryRow(`
		SELECT id, password, role, tenant_id
		FROM users 
		WHERE email = $1
	`, req.Email).Scan(&userID, &hashedPassword, &role, &tenantID)

	log.Println("Stored hash:", hashedPassword)
	log.Println("Input password:", req.Password)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// compare password
	if err := utils.CheckPassword(req.Password, hashedPassword); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// generate JWT
	token, err := utils.GenerateToken(userID, role, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"role":      role,
		"tenant_id": tenantID.Int32,
	})
}

// register coach(admin-only)

type RegisterCoachRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) RegisterCoach(c *gin.Context) {
	var req RegisterCoachRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Get admin's tenant ID
	userID := c.GetInt("user_id")
	var tenantID sql.NullInt32
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id = $1", userID).Scan(&tenantID)
	if err != nil || !tenantID.Valid {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin tenant not found"})
		return
	}

	// check if email already exists
	var exists bool
	err = h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		req.Email,
	).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already registered"})
		return
	}

	// hash password
	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}

	// create user with role=coach
	var newUserID int
	err = h.DB.QueryRow(`
		INSERT INTO users (tenant_id, email, password, role)
		VALUES ($1, $2, $3, 'coach')
		RETURNING id
	`, tenantID.Int32, req.Email, hashed).Scan(&newUserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// create corresponding coach record
	var coachID int
	err = h.DB.QueryRow(`
		INSERT INTO coaches (tenant_id, user_id, name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, tenantID.Int32, newUserID, req.Name).Scan(&coachID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "coach profile creation failed"})
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

// update password

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// fetch current hashed password
	var currentHash string
	err := h.DB.QueryRow(
		"SELECT password FROM users WHERE id = $1",
		userID,
	).Scan(&currentHash)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// verify current password
	if err := utils.CheckPassword(req.CurrentPassword, currentHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect current password"})
		return
	}

	// hash new password
	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}

	// update
	_, err = h.DB.Exec(
		"UPDATE users SET password = $1 WHERE id = $2",
		newHash, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

// google oauth

type GoogleRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req GoogleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	payload, err := idtoken.Validate(context.Background(), req.IDToken, "")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google token"})
		return
	}

	email := payload.Claims["email"].(string)

	var userID int
	var role string
	var tenantID sql.NullInt32

	err = h.DB.QueryRow(`
		SELECT id, role, tenant_id FROM users WHERE email = $1
	`, email).Scan(&userID, &role, &tenantID)

	if err == sql.ErrNoRows {
		// Auto-create a new tenant and admin account
		name, _ := payload.Claims["name"].(string)
		if name == "" {
			name = "New Organization"
		} else {
			name = name + "'s Organization"
		}

		// Create tenant
		var newTenantID int
		err = h.DB.QueryRow(`
			INSERT INTO tenants (name) VALUES ($1) RETURNING id
		`, name).Scan(&newTenantID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
			return
		}

		// Create admin user
		err = h.DB.QueryRow(`
			INSERT INTO users (tenant_id, email, role)
			VALUES ($1, $2, 'admin')
			RETURNING id
		`, newTenantID, email).Scan(&userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin account"})
			return
		}
		role = "admin"
		tenantID = sql.NullInt32{Int32: int32(newTenantID), Valid: true}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Restrict to admins and super_admins
	if role == "coach" || role == "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Google login is restricted to organization owners."})
		return
	}

	// generate JWT
	token, _ := utils.GenerateToken(userID, role, 0)

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"role":      role,
		"tenant_id": tenantID.Int32,
	})
}


