package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	student := r.Group("/student")
	{
		student.POST("/login", handlers.StudentLogin)
		student.POST("/submit", handlers.SubmitAnswers)
	}
	return r
}
