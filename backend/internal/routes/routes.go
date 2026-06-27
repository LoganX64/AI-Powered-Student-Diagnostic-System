package routes

import (
	"ai-student-diagnostic/backend/internal/auth"
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/repository"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// Global error handlers
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	})

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

	// health check
	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Initialize repos
	userRepo := repository.NewUserRepo(db)
	studentRepo := repository.NewStudentRepo(db)
	coachRepo := repository.NewCoachRepo(db)
	testRepo := repository.NewTestRepo(db)
	assignmentRepo := repository.NewAssignmentRepo(db)
	attemptRepo := repository.NewAttemptRepo(db)

	// Initialize handlers
	authHandler := auth.NewAuthHandler(db)
	adminHandler := handlers.NewAdminHandler(userRepo, studentRepo, coachRepo, testRepo, assignmentRepo, attemptRepo)
	coachHandler := handlers.NewCoachHandler(userRepo, studentRepo, coachRepo, testRepo, assignmentRepo, attemptRepo)
	studentHandler := handlers.NewStudentHandler(studentRepo, assignmentRepo, attemptRepo, testRepo)

	// auth
	authRoute := r.Group("/auth")
	{
		authRoute.POST("/login", authHandler.UserLogin)
		authRoute.POST("/register-admin", authHandler.RegisterAdmin)
		authRoute.POST("/google", authHandler.GoogleLogin)
	}

	// student
	student := r.Group("/student")
	{
		student.POST("/login", studentHandler.StudentLogin)

		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware(db))
		{
			protected.POST("/submit/:id", studentHandler.SubmitAnswers)
			protected.GET("/assignments", studentHandler.ListStudentAssignments)
			protected.GET("/assignments/:id/questions", studentHandler.GetAssignmentQuestions)
		}
	}

	// admin
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
		admin.GET("/students/:id/assignments/:assignmentId", adminHandler.GetAssignmentResults)
		admin.DELETE("/students/:id", adminHandler.DeleteStudent)
		admin.PUT("/students/:id/reactivate", adminHandler.ReactivateStudent)
		admin.GET("/coaches", adminHandler.ListCoaches)
		admin.GET("/coaches/:id", adminHandler.GetCoach)
		admin.DELETE("/coaches/:id", adminHandler.DeleteCoach)
		admin.PUT("/coaches/:id/reactivate", adminHandler.ReactivateCoach)
		admin.GET("/coaches/:id/tests", adminHandler.ListCoachTests)
		admin.GET("/coaches/:id/students", adminHandler.ListCoachStudents)
		admin.GET("/subjects", adminHandler.ListSubjects)
		admin.GET("/assignments", adminHandler.ListAssignments)
		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
		admin.GET("/students/:id/subjects/:subject_id/results", adminHandler.GetStudentSubjectResults)
	}

	// coach
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
		coach.GET("/students/:id/assignments/:assignmentId", coachHandler.GetAssignmentResults)
		coach.DELETE("/students/:id", coachHandler.DeleteStudent)
		coach.PUT("/students/:id/reactivate", coachHandler.ReactivateStudent)
		coach.GET("/subjects", coachHandler.ListSubjects)
		coach.GET("/assignments", coachHandler.ListAssignments)

		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
