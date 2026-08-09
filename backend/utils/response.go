package utils

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// SafeErrorResponse logs the real error and returns a safe response.
// In DEBUG mode: returns raw error for frontend debugging.
// Otherwise: returns generic message.
func SafeErrorResponse(c *gin.Context, status int, err error, message string) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	if os.Getenv("DEBUG") == "true" {
		c.JSON(status, gin.H{"error": err.Error()})
	} else {
		c.JSON(status, gin.H{"error": message})
	}
}

func BadRequest(c *gin.Context, msg string) {
	log.Printf("[400] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func Unauthorized(c *gin.Context, msg string) {
	log.Printf("[401] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
}

func Forbidden(c *gin.Context, msg string) {
	log.Printf("[403] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusForbidden, gin.H{"error": msg})
}

func NotFound(c *gin.Context, msg string) {
	log.Printf("[404] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}

func Conflict(c *gin.Context, msg string) {
	log.Printf("[409] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusConflict, gin.H{"error": msg})
}

func InternalError(c *gin.Context, err error, msg string) {
	log.Printf("[500] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	if os.Getenv("DEBUG") == "true" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
	}
}
