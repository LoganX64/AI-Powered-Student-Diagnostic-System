package middleware

import (
	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")

		if !exists {
			utils.Unauthorized(c, "role missing")
			c.Abort()
			return
		}

		role, ok := roleVal.(string)
		if !ok {
			utils.Unauthorized(c, "invalid user role")
			c.Abort()
			return
		}

		for _, allowed := range allowedRoles {
			if role == allowed {
				c.Next()
				return
			}
		}

		utils.Forbidden(c, "access denied")
		c.Abort()
	}
}
