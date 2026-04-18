package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// AUTH (admin + coach)
	authHandler := handlers.NewAuthHandler(db)
	auth := r.Group("/auth")
	{
		auth.POST("/login", authHandler.UserLogin)
		auth.POST("/register", authHandler.Register)
		auth.POST("/google", authHandler.GoogleLogin)
	}

	// STUDENT
	student := r.Group("/student")
	{
		//
		student.POST("/login", handlers.StudentLogin)

		// protected routes
		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/submit", handlers.SubmitAnswers)
		}
	}

	// ADMIN
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
