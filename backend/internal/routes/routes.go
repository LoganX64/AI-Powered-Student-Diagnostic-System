package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()
	student := r.Group("/student")
	{
		student.POST("/login", handlers.StudentLogin)
		auth := student.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.POST("/submit", handlers.SubmitAnswers)
		}
	}
	adminHandler := handlers.NewAdminHandler(db)

	admin := r.Group("/admin")
	{
		admin.POST("/students", adminHandler.CreateStudent)
		admin.POST("/coaches", adminHandler.CreateCoach)
		admin.POST("/subjects", adminHandler.CreateSubject)
		admin.POST("/tests", adminHandler.CreateTest)
		admin.POST("/questions", adminHandler.CreateQuestion)
		admin.POST("/assignments", adminHandler.CreateAssignment)

		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
	}
	return r
}
