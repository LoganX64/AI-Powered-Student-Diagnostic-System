package utils

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// SafeErrorResponse logs the real error and returns a safe response.
// In production: returns generic message.
// In development: returns raw error for frontend debugging.
func SafeErrorResponse(c *gin.Context, status int, err error, message string) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	if os.Getenv("APP_ENV") == "production" {
		c.JSON(status, gin.H{"error": message})
	} else {
		c.JSON(status, gin.H{"error": err.Error()})
	}
}
