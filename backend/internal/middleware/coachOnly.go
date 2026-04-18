package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CoachOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")

		if role != "coach" {
			c.JSON(http.StatusForbidden, gin.H{"error": "coach access only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
