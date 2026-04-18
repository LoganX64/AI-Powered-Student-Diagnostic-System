package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func StudentOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")

		if role != "student" {
			c.JSON(http.StatusForbidden, gin.H{"error": "student access only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
