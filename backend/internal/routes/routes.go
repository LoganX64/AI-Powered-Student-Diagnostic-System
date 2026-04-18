package routes

import (
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	adminHandler := handlers.NewAdminHandler(db)

	// =========================
	// PUBLIC ROUTES
	// =========================
	authHandler := handlers.NewAuthHandler(db)
	r.POST("/auth/login", authHandler.UserLogin)
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/google", authHandler.GoogleLogin)

	// =========================
	// PROTECTED ROUTES (JWT)
	// =========================
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())

	// =========================
	// STUDENT ROUTES
	// =========================
	student := r.Group("/student")
	student.Use(middleware.StudentOnly())
	{
		student.POST("/submit", handlers.SubmitAnswers)
	}
	// =========================
	// ADMIN ROUTES
	// =========================
	admin := auth.Group("/admin")
	admin.Use(middleware.AdminOnly())
	{
		admin.POST("/students", adminHandler.CreateStudent)
		admin.POST("/coaches", adminHandler.CreateCoach)
		admin.POST("/subjects", adminHandler.CreateSubject)
		admin.POST("/tests", adminHandler.CreateTest)
		admin.POST("/questions", adminHandler.CreateQuestion)
		admin.POST("/assignments", adminHandler.CreateAssignment)

		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
	}

	// =========================
	// COACH ROUTES (future ready)
	// =========================
	coach := auth.Group("/coach")
	coach.Use(middleware.CoachOnly())
	{
		// example:
		// coach.GET("/my-tests", ...)
		// coach.POST("/assign", ...)
	}

	return r
}
