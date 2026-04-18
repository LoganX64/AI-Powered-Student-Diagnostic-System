package handlers

import (
	"ai-student-diagnostic/backend/utils"
	"context"
	"database/sql"
	"net/http"

	"google.golang.org/api/idtoken"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	err := h.DB.QueryRow(`
		SELECT id, password, role 
		FROM users 
		WHERE email = $1
	`, req.Email).Scan(&userID, &hashedPassword, &role)

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
		"token": token,
		"role":  role,
	})
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"` // admin / coach
	Name     string `json:"name"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if req.Role != "admin" && req.Role != "coach" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	// hash password
	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}

	var userID int
	err = h.DB.QueryRow(`
		INSERT INTO users (email, password, role)
		VALUES ($1,$2,$3)
		RETURNING id
	`, req.Email, hashed, req.Role).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// if coach → create coach row
	if req.Role == "coach" {
		_, err = h.DB.Exec(`
			INSERT INTO coaches (user_id, name)
			VALUES ($1,$2)
		`, userID, req.Name)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "coach creation failed"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
	})
}

// google Oauth

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

	err = h.DB.QueryRow(`
		SELECT id, role FROM users WHERE email = $1
	`, email).Scan(&userID, &role)

	// if not exist → create coach
	if err == sql.ErrNoRows {
		err = h.DB.QueryRow(`
			INSERT INTO users (email, role)
			VALUES ($1,'coach')
			RETURNING id
		`, email).Scan(&userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user creation failed"})
			return
		}

		_, _ = h.DB.Exec(`
			INSERT INTO coaches (user_id, name)
			VALUES ($1,$2)
		`, userID, payload.Claims["name"])

		role = "coach"
	}

	// generate JWT
	token, _ := utils.GenerateToken(userID, role, 0)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"role":  role,
	})
}
