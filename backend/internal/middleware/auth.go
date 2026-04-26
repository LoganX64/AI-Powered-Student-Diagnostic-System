package middleware

import (
	"ai-student-diagnostic/backend/utils"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[AUTH MIDDLEWARE] Path: %s, Method: %s\n", c.Request.URL.Path, c.Request.Method)

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			log.Printf("[AUTH MIDDLEWARE] No Authorization header\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		log.Printf("[AUTH MIDDLEWARE] Token received: %s...\n", tokenStr[:20])

		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			log.Printf("[AUTH MIDDLEWARE] Token validation failed: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		log.Printf("[AUTH MIDDLEWARE] Token valid for user %d with role %s\n", claims.UserID, claims.Role)

		// 2. Database Verify (Ensures token is invalidated if user is deleted/DB reset)
		if claims.Role == "student" {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)", claims.StudentID).Scan(&exists)
			if err != nil || !exists {
				log.Printf("[AUTH MIDDLEWARE] Student does not exist\n")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "student no longer exists"})
				c.Abort()
				return
			}
			c.Set("student_id", claims.StudentID)
		} else {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", claims.UserID).Scan(&exists)
			if err != nil || !exists {
				log.Printf("[AUTH MIDDLEWARE] User does not exist\n")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user no longer exists"})
				c.Abort()
				return
			}
			c.Set("user_id", claims.UserID)
		}

		c.Set("role", claims.Role)
		log.Printf("[AUTH MIDDLEWARE] Auth passed for user %d\n", claims.UserID)
		c.Next()
	}
}
