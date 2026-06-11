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

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

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
			protected.POST("/submit/:id", handlers.SubmitAnswers)
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
		admin.POST("/tests/:id/questions", adminHandler.CreateQuestion)
		admin.POST("/assignments", adminHandler.CreateAssignment)

		admin.GET("/tests", adminHandler.ListTests)
		admin.GET("/tests/:id", adminHandler.GetTest)
		admin.GET("/tests/:id/questions", adminHandler.GetTestQuestions)
		admin.GET("/students", adminHandler.ListStudents)
		admin.GET("/coaches", adminHandler.ListCoaches)
		admin.GET("/subjects", adminHandler.ListSubjects)
		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
		admin.GET("/students/:id/subjects/:subject_id/results", adminHandler.GetStudentSubjectResults)
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
		coach.POST("/tests/:id/questions", coachHandler.CreateQuestion)
		coach.POST("/assignments", coachHandler.CreateAssignment)
		coach.POST("/subjects", adminHandler.CreateSubject)

		coach.GET("/tests", coachHandler.ListTests)
		coach.GET("/students", coachHandler.ListStudents)
		coach.GET("/subjects", coachHandler.ListSubjects)

		// update own password
		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
