package middleware

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(studentRepo *repository.StudentRepo, userRepo *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			utils.Unauthorized(c, "missing token")
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Unauthorized(c, "invalid token format")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			utils.Unauthorized(c, "missing token")
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			utils.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}

		if claims.Role == "student" {
			exists, err := studentRepo.ExistsByID(claims.StudentID)
			if err != nil {
				utils.InternalError(c, err, "service temporarily unavailable")
				c.Abort()
				return
			}
			if !exists {
				utils.Unauthorized(c, "student no longer exists")
				c.Abort()
				return
			}
			c.Set("student_id", claims.StudentID)
		} else {
			exists, err := userRepo.ExistsByID(claims.UserID)
			if err != nil {
				utils.InternalError(c, err, "service temporarily unavailable")
				c.Abort()
				return
			}
			if !exists {
				utils.Unauthorized(c, "user no longer exists")
				c.Abort()
				return
			}
			c.Set("user_id", claims.UserID)
		}

		c.Set("role", claims.Role)
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}

func AuthMiddlewareWS(studentRepo *repository.StudentRepo, userRepo *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			utils.Unauthorized(c, "missing token")
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			utils.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}

		if claims.Role == "student" {
			exists, err := studentRepo.ExistsByID(claims.StudentID)
			if err != nil {
				utils.InternalError(c, err, "service temporarily unavailable")
				c.Abort()
				return
			}
			if !exists {
				utils.Unauthorized(c, "student no longer exists")
				c.Abort()
				return
			}
			c.Set("student_id", claims.StudentID)
		} else {
			exists, err := userRepo.ExistsByID(claims.UserID)
			if err != nil {
				utils.InternalError(c, err, "service temporarily unavailable")
				c.Abort()
				return
			}
			if !exists {
				utils.Unauthorized(c, "user no longer exists")
				c.Abort()
				return
			}
			c.Set("user_id", claims.UserID)
		}

		c.Set("role", claims.Role)
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}
