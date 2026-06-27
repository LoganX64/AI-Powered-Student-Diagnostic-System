package middleware

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(studentRepo *repository.StudentRepo, userRepo *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims.Role == "student" {
			exists, err := studentRepo.ExistsByID(claims.StudentID)
			if err != nil || !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "student no longer exists"})
				c.Abort()
				return
			}
			c.Set("student_id", claims.StudentID)
		} else {
			exists, err := userRepo.ExistsByID(claims.UserID)
			if err != nil || !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user no longer exists"})
				c.Abort()
				return
			}
			c.Set("user_id", claims.UserID)
		}

		c.Set("role", claims.Role)
		c.Next()
	}
}
