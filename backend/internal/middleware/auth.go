package middleware

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"fmt"

	"github.com/gin-gonic/gin"
)

// roles names the cookie each accepted session role is read from. The role is
// also embedded in the JWT itself, so accepting several roles (e.g. the shared
// /auth/profile group) simply tries each cookie. When several are valid the
// X-Role header — which the frontend sets from the current route — wins.
func extractToken(c *gin.Context, roles []string) string {
	preferred := c.GetHeader("X-Role")
	try := func(role string) string {
		if t, err := c.Cookie(role + "_token"); err == nil {
			return t
		}
		return ""
	}
	if preferred != "" {
		for _, r := range roles {
			if r == preferred {
				if t := try(r); t != "" {
					return t
				}
			}
		}
	}
	for _, r := range roles {
		if t := try(r); t != "" {
			return t
		}
	}
	return ""
}

func AuthMiddleware(studentRepo *repository.StudentRepo, userRepo *repository.UserRepo, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c, roles)
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

		if len(roles) > 0 {
			allowed := false
			for _, r := range roles {
				if r == claims.Role {
					allowed = true
					break
				}
			}
			if !allowed {
				utils.Unauthorized(c, "wrong session role")
				c.Abort()
				return
			}
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

func VideoTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The HTML <video> element cannot set headers or WS subprotocols, so the
		// short-lived video token travels as a query parameter.
		tokenStr := c.Query("token")
		if tokenStr == "" {
			utils.Unauthorized(c, "missing token")
			c.Abort()
			return
		}

		claims, err := utils.ValidateVideoToken(tokenStr)
		if err != nil {
			utils.Unauthorized(c, "invalid or expired video token")
			c.Abort()
			return
		}

		// Ensure the token's assignment_id matches the route param.
		assignmentID := c.Param("id")
		if assignmentID == "" || fmt.Sprintf("%d", claims.AssignmentID) != assignmentID {
			utils.Forbidden(c, "video token does not match assignment")
			c.Abort()
			return
		}

		c.Set("assignment_id", claims.AssignmentID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
