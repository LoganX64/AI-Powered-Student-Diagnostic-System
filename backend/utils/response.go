package utils

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// errorBody builds the standard error response body and merges any extra
// structured fields (e.g. deactivated_id, position) on top of {"error": msg}.
func errorBody(msg string, extra ...gin.H) gin.H {
	body := gin.H{"error": msg}
	for _, e := range extra {
		for k, v := range e {
			body[k] = v
		}
	}
	return body
}

// SafeErrorResponse logs the real error and returns a safe response.
// In DEBUG mode: returns raw error for frontend debugging.
// Otherwise: returns generic message.
func SafeErrorResponse(c *gin.Context, status int, err error, message string, extra ...gin.H) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	if os.Getenv("DEBUG") == "true" {
		c.JSON(status, errorBody(err.Error(), extra...))
	} else {
		c.JSON(status, errorBody(message, extra...))
	}
}

func BadRequest(c *gin.Context, msg string, extra ...gin.H) {
	log.Printf("[400] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusBadRequest, errorBody(msg, extra...))
}

func Unauthorized(c *gin.Context, msg string, extra ...gin.H) {
	log.Printf("[401] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusUnauthorized, errorBody(msg, extra...))
}

func Forbidden(c *gin.Context, msg string, extra ...gin.H) {
	log.Printf("[403] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusForbidden, errorBody(msg, extra...))
}

func NotFound(c *gin.Context, msg string, extra ...gin.H) {
	log.Printf("[404] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusNotFound, errorBody(msg, extra...))
}

func Conflict(c *gin.Context, msg string, extra ...gin.H) {
	log.Printf("[409] %s %s: %s", c.Request.Method, c.Request.URL.Path, msg)
	c.JSON(http.StatusConflict, errorBody(msg, extra...))
}

func InternalError(c *gin.Context, err error, msg string, extra ...gin.H) {
	log.Printf("[500] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	if os.Getenv("DEBUG") == "true" {
		c.JSON(http.StatusInternalServerError, errorBody(err.Error(), extra...))
	} else {
		c.JSON(http.StatusInternalServerError, errorBody(msg, extra...))
	}
}
