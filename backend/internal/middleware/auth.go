package middleware

import (
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// IMPORTANT: always set role
		c.Set("role", claims.Role)

		// differentiate user vs student
		if claims.Role == "student" {
			c.Set("student_id", claims.StudentID)
		} else {
			c.Set("user_id", claims.UserID)
		}

		c.Next()
	}
}
