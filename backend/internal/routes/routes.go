package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// ================= AUTH =================
	authHandler := handlers.NewAuthHandler(db)

	auth := r.Group("/auth")
	{
		auth.POST("/login", authHandler.UserLogin)    // admin + coach
		auth.POST("/google", authHandler.GoogleLogin) // coach via google
	}

	// ================= STUDENT =================
	student := r.Group("/student")
	{
		// public
		student.POST("/login", handlers.StudentLogin)

		// protected
		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/submit", handlers.SubmitAnswers)
		}
	}

	// ================= ADMIN =================
	adminHandler := handlers.NewAdminHandler(db)

	admin := r.Group("/admin")
	admin.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("admin"),
	)
	{
		// create a coach account (login credentials + coach profile)
		admin.POST("/register", authHandler.Register)

		admin.POST("/subjects", adminHandler.CreateSubject)
		admin.POST("/tests", adminHandler.CreateTest)
		admin.POST("/questions", adminHandler.CreateQuestion)

		admin.POST("/assignments", adminHandler.CreateAssignment)

		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
	}

	// ================= COACH =================
	coachHandler := handlers.NewAdminHandler(db)

	coach := r.Group("/coach")
	coach.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("coach"),
	)
	{
		coach.GET("/students/:id/sqi", coachHandler.GetStudentSQI)

		coach.POST("/students", coachHandler.CreateStudent)
		coach.POST("/tests", coachHandler.CreateTest)
		coach.POST("/questions", coachHandler.CreateQuestion)
		coach.POST("/assignments", coachHandler.CreateAssignment)

		// update own password
		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
