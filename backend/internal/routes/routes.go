package routes

import (
	"ai-student-diagnostic/backend/internal/auth"
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	//  auth
	authHandler := auth.NewAuthHandler(db)

	authRoute := r.Group("/auth")
	{
		authRoute.POST("/login", authHandler.UserLogin)
		authRoute.POST("/register-admin", authHandler.RegisterAdmin)
		authRoute.POST("/google", authHandler.GoogleLogin)
	}

	//  student
	student := r.Group("/student")
	{
		// public
		student.POST("/login", handlers.StudentLogin)

		// protected
		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware(db))
		{
			protected.POST("/submit", handlers.SubmitAnswers)
		}
	}

	//  admin
	adminHandler := handlers.NewAdminHandler(db)

	admin := r.Group("/admin")
	admin.Use(
		middleware.AuthMiddleware(db),
		middleware.RoleMiddleware("admin"),
	)
	{
		admin.POST("/register-coach", authHandler.RegisterCoach)

		admin.POST("/subjects", adminHandler.CreateSubject)
		admin.POST("/students", adminHandler.CreateStudent)
		admin.POST("/tests", adminHandler.CreateTest)
		admin.POST("/questions", adminHandler.CreateQuestion)
		admin.POST("/assignments", adminHandler.CreateAssignment)
		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
	}

	//  coach
	coachHandler := handlers.NewCoachHandler(db)

	coach := r.Group("/coach")
	coach.Use(
		middleware.AuthMiddleware(db),
		middleware.RoleMiddleware("coach"),
	)
	{
		coach.GET("/students/:id/sqi", coachHandler.GetStudentSQI)

		coach.POST("/students", coachHandler.CreateStudent)
		coach.POST("/tests", coachHandler.CreateTest)
		coach.POST("/questions", coachHandler.CreateQuestion)
		coach.POST("/assignments", coachHandler.CreateAssignment)
		coach.POST("/subjects", adminHandler.CreateSubject)

		// update own password
		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
