package routes

import (
	"ai-student-diagnostic/backend/internal/auth"
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB, cfg *config.Config, allowedOrigins []string, trustedProxies []string) *gin.Engine {
	r := gin.Default()

	if len(trustedProxies) > 0 {
		if err := r.SetTrustedProxies(trustedProxies); err != nil {
			log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
		}
	} else {
		r.SetTrustedProxies(nil)
	}

	r.NoRoute(func(c *gin.Context) {
		utils.NotFound(c, "route not found")
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	})

	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, o := range allowedOrigins {
			if o == origin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			log.Printf("[HEALTH] DB ping failed: %v", err)
			c.JSON(503, gin.H{"status": "unhealthy", "error": "database unreachable"})
			return
		}
		c.JSON(200, gin.H{"status": "healthy"})
	})

	userRepo := repository.NewUserRepo(db)
	studentRepo := repository.NewStudentRepo(db)
	coachRepo := repository.NewCoachRepo(db)
	testPaperRepo := repository.NewTestPaperRepo(db)
	assignmentRepo := repository.NewAssignmentRepo(db)
	attemptRepo := repository.NewAttemptRepo(db)
	loginAttemptRepo := repository.NewLoginAttemptRepo(db)
	batchRepo := repository.NewBatchRepo(db)

	attemptService := services.NewAttemptService(attemptRepo, assignmentRepo, studentRepo, testPaperRepo)
	assignmentService := services.NewAssignmentService(assignmentRepo, studentRepo, testPaperRepo, coachRepo, userRepo)
	authService := services.NewAuthService(userRepo, coachRepo)

	authHandler := auth.NewAuthHandler(authService, loginAttemptRepo)
	adminHandler := handlers.NewAdminHandler(userRepo, studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, attemptService, assignmentService, cfg)
	coachHandler := handlers.NewCoachHandler(studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, attemptService, assignmentService, cfg)
	studentHandler := handlers.NewStudentHandler(studentRepo, assignmentRepo, attemptRepo, testPaperRepo, attemptService, loginAttemptRepo, cfg)

	authRoute := r.Group("/auth")
	authRoute.Use(middleware.RateLimit())
	{
		authRoute.POST("/login", authHandler.UserLogin)
		authRoute.POST("/register-admin", authHandler.RegisterAdmin)
	}

	student := r.Group("/student")
	{
		student.POST("/login", middleware.RateLimit(), studentHandler.StudentLogin)

		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware(studentRepo, userRepo))
		{
			protected.POST("/submit/:id", studentHandler.SubmitAnswers)
			protected.GET("/assignments", studentHandler.ListStudentAssignments)
			protected.GET("/assignments/:id/questions", studentHandler.GetAssignmentQuestions)
		}
	}

	admin := r.Group("/admin")
	admin.Use(
		middleware.AuthMiddleware(studentRepo, userRepo),
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
		admin.DELETE("/subjects/:id", adminHandler.DeleteSubject)
		admin.PUT("/subjects/:id/reactivate", adminHandler.ReactivateSubject)
		admin.GET("/assignments", adminHandler.ListAssignments)
		admin.GET("/students/:id/sqi", adminHandler.GetStudentSQI)
		admin.POST("/students/sqi-batch", adminHandler.GetStudentSQIBatch)
		admin.POST("/coaches/stats-batch", adminHandler.GetCoachStatsBatch)

		admin.POST("/batches", adminHandler.CreateBatch)
		admin.GET("/batches", adminHandler.ListBatches)
		admin.DELETE("/batches/:id", adminHandler.DeleteBatch)
		admin.PATCH("/students/:id/batch", adminHandler.TransferStudentBatch)
	}

	coach := r.Group("/coach")
	coach.Use(
		middleware.AuthMiddleware(studentRepo, userRepo),
		middleware.RoleMiddleware("coach"),
	)
	{
		coach.GET("/students/:id/sqi", coachHandler.GetStudentSQI)
		coach.POST("/students/sqi-batch", adminHandler.GetStudentSQIBatch)

		coach.POST("/students", coachHandler.CreateStudent)
		coach.POST("/tests", coachHandler.CreateTest)
		coach.POST("/tests/:id/questions", adminHandler.CreateQuestion)
		coach.POST("/assignments", coachHandler.CreateAssignment)
		coach.POST("/subjects", adminHandler.CreateSubject)

		coach.PUT("/tests/:id", adminHandler.UpdateTest)
		coach.PUT("/tests/:id/questions/:qid", adminHandler.UpdateQuestion)

		coach.DELETE("/tests/:id", adminHandler.DeleteTest)
		coach.DELETE("/tests/:id/questions/:qid", adminHandler.DeleteQuestion)

		coach.GET("/tests/:id", adminHandler.GetTest)
		coach.GET("/tests", adminHandler.ListTests)
		coach.GET("/tests/:id/questions", adminHandler.GetTestQuestions)
		coach.GET("/students", adminHandler.ListStudents)
		coach.GET("/students/:id", adminHandler.GetStudent)
		coach.GET("/students/:id/assignments", adminHandler.ListStudentAssignments)
		coach.GET("/students/:id/assignments/:assignmentId", coachHandler.GetAssignmentResults)
		coach.DELETE("/students/:id", adminHandler.DeleteStudent)
		coach.PUT("/students/:id/reactivate", adminHandler.ReactivateStudent)
		coach.GET("/subjects", adminHandler.ListSubjects)
		coach.DELETE("/subjects/:id", adminHandler.DeleteSubject)
		coach.PUT("/subjects/:id/reactivate", adminHandler.ReactivateSubject)
		coach.GET("/assignments", adminHandler.ListAssignments)

		coach.POST("/batches", coachHandler.CreateBatch)
		coach.GET("/batches", coachHandler.ListBatches)
		coach.DELETE("/batches/:id", coachHandler.DeleteBatch)
		coach.PATCH("/students/:id/batch", coachHandler.TransferStudentBatch)

		coach.PUT("/password", authHandler.UpdatePassword)
	}

	return r
}
