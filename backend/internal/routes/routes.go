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
			protected.GET("/assignments", handlers.ListStudentAssignments)
			protected.GET("/assignments/:id/questions", handlers.GetAssignmentQuestions)
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

		admin.PUT("/tests/:id", adminHandler.UpdateTest)
		admin.PUT("/tests/:id/questions/:qid", adminHandler.UpdateQuestion)

		admin.DELETE("/tests/:id", adminHandler.DeleteTest)
		admin.DELETE("/tests/:id/questions/:qid", adminHandler.DeleteQuestion)

		admin.GET("/tests", adminHandler.ListTests)
		admin.GET("/tests/:id", adminHandler.GetTest)
		admin.GET("/tests/:id/questions", adminHandler.GetTestQuestions)
		admin.GET("/students", adminHandler.ListStudents)
		admin.GET("/students/:id", adminHandler.GetStudent)
		admin.GET("/students/:id/assignments", adminHandler.ListStudentAssignments)
		admin.DELETE("/students/:id", adminHandler.DeleteStudent)
		admin.GET("/coaches", adminHandler.ListCoaches)
		admin.GET("/coaches/:id", adminHandler.GetCoach)
		admin.GET("/coaches/:id/tests", adminHandler.ListCoachTests)
		admin.GET("/coaches/:id/students", adminHandler.ListCoachStudents)
		admin.GET("/subjects", adminHandler.ListSubjects)
		admin.GET("/assignments", adminHandler.ListAssignments)
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

		coach.PUT("/tests/:id", adminHandler.UpdateTest)
		coach.PUT("/tests/:id/questions/:qid", adminHandler.UpdateQuestion)

		coach.DELETE("/tests/:id", adminHandler.DeleteTest)
		coach.DELETE("/tests/:id/questions/:qid", adminHandler.DeleteQuestion)

		coach.GET("/tests", coachHandler.ListTests)
		coach.GET("/tests/:id/questions", adminHandler.GetTestQuestions)
		coach.GET("/students", coachHandler.ListStudents)
		coach.GET("/students/:id", coachHandler.GetStudent)
		coach.GET("/students/:id/assignments", coachHandler.ListStudentAssignments)
		coach.DELETE("/students/:id", coachHandler.DeleteStudent)
		coach.GET("/subjects", coachHandler.ListSubjects)
		coach.GET("/assignments", coachHandler.ListAssignments)

		// update own password
		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
