package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"
	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()
	student := r.Group("/student")
	{
		student.POST("/login", handlers.StudentLogin)
		student.POST("/submit", handlers.SubmitAnswers)
	}
	adminHandler := handlers.NewAdminHandler(db)

	admin := r.Group("/admin")
	{
		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
	}
	return r
}
