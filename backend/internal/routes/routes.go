package routes

import (
	"ai-student-diagnostic/backend/internal/auth"
	"ai-student-diagnostic/backend/internal/cache"
	"ai-student-diagnostic/backend/internal/config"
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/utils"
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB, cfg *config.Config, allowedOrigins []string, trustedProxies []string) (*gin.Engine, func() error) {
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
					c.Header("Vary", "Origin")
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
	jobRepo := repository.NewJobRepo(db)

	attemptService := services.NewAttemptService(attemptRepo, assignmentRepo, studentRepo, testPaperRepo)
	assignmentService := services.NewAssignmentService(assignmentRepo, studentRepo, testPaperRepo, coachRepo, userRepo, cfg.RedisEnabled)
	jobService := services.NewJobService(jobRepo, attemptService, cfg.ComputeChunkSize)
	authService := services.NewAuthService(userRepo, coachRepo)

	redisClient := cache.NewRedis(cfg)

	// Storage backend for the video proctoring tier
	var storageBackend storage.Storage
	if cfg.CloudinaryURL != "" {
		if cs, err := storage.NewCloudinary(cfg.CloudinaryURL); err == nil {
			storageBackend = cs
		} else {
			log.Printf("[STORAGE] cloudinary init failed, falling back to local: %v", err)
		}
	}
	if storageBackend == nil {
		storageBackend = storage.NewLocal(cfg.UploadDir)
	}

	// Autosave buffer (Redis-backed batched flush) only in scale mode.
	var autosaveBuffer *services.AutosaveBuffer
	if cfg.RedisEnabled && redisClient != nil {
		autosaveBuffer = services.NewAutosaveBuffer(redisClient, attemptRepo)
		autosaveBuffer.Start()
	}

	jobQueue := queue.New(cfg)

	authHandler := auth.NewAuthHandler(authService, loginAttemptRepo)
	adminHandler := handlers.NewAdminHandler(userRepo, studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, jobRepo, attemptService, assignmentService, jobService, jobQueue, cfg)
	coachHandler := handlers.NewCoachHandler(studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, jobRepo, attemptService, assignmentService, jobService, jobQueue, cfg)
	studentHandler := handlers.NewStudentHandler(studentRepo, assignmentRepo, attemptRepo, testPaperRepo, attemptService, loginAttemptRepo, jobQueue, autosaveBuffer, storageBackend, cfg)

	authRoute := r.Group("/auth")
	authRoute.Use(middleware.RateLimit())
	{
		authRoute.POST("/login", authHandler.UserLogin)
		authRoute.POST("/register-admin", authHandler.RegisterAdmin)
	}

	student := r.Group("/student")
	{
		student.POST("/login", middleware.RateLimit(), studentHandler.StudentLogin)

		// Shared (Redis) rate limiter when available, else per-instance in-memory.
		limiter := middleware.NewRateLimiter(redisClient)

		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware(studentRepo, userRepo))
		{
			protected.POST("/submit/:id", studentHandler.SubmitAnswers)
			protected.GET("/assignments", studentHandler.ListStudentAssignments)
			protected.GET("/assignments/:id/questions", studentHandler.GetAssignmentQuestions)
			protected.POST("/assignments/:id/start", studentHandler.StartExam)
			protected.POST("/assignments/:id/autosave", limiter, studentHandler.Autosave)
			protected.GET("/assignments/:id/state", studentHandler.GetState)
			protected.POST("/assignments/:id/submit", limiter, studentHandler.SubmitExam)
			protected.POST("/assignments/:id/video-chunk", limiter, studentHandler.VideoChunk)
			protected.POST("/api/time", studentHandler.ServerTime)
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
		admin.POST("/assignments/batch", adminHandler.CreateBatchAssignment)

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

		admin.POST("/sqi/compute", adminHandler.ComputeSQI)
		admin.POST("/sqi/compute-batch", adminHandler.ComputeSQIBatch)
		admin.GET("/jobs/:id", adminHandler.GetJob)
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
		coach.POST("/assignments/batch", coachHandler.CreateBatchAssignment)
		coach.POST("/subjects", adminHandler.CreateSubject)

		coach.PUT("/tests/:id", adminHandler.UpdateTest)
		coach.PUT("/tests/:id/questions/:qid", adminHandler.UpdateQuestion)

		coach.DELETE("/tests/:id", adminHandler.DeleteTest)
		coach.DELETE("/tests/:id/questions/:qid", adminHandler.DeleteQuestion)

		coach.GET("/tests/:id", adminHandler.GetTest)
		coach.GET("/tests", adminHandler.ListTests)
		coach.GET("/tests/:id/questions", adminHandler.GetTestQuestions)
		coach.GET("/students", adminHandler.ListStudents)
		coach.GET("/coaches", coachHandler.ListCoaches)
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

		coach.POST("/sqi/compute", coachHandler.ComputeSQI)
		coach.POST("/sqi/compute-batch", coachHandler.ComputeSQIBatch)
		coach.GET("/jobs/:id", coachHandler.GetJob)

		coach.PUT("/password", authHandler.UpdatePassword)
	}

	jobQueue.Start(
		func(jobID, tenantID int) error { return jobService.Process(jobID, tenantID) },
		func(p queue.FinalizePayload) error {
			answers := make([]services.AnswerInput, len(p.Answers))
			for i, a := range p.Answers {
				answers[i] = services.AnswerInput{
					QuestionID:        a.QuestionID,
					SelectedAnswer:    a.SelectedAnswer,
					TimeSpent:         a.TimeSpent,
					Seen:              a.Seen,
					MarkedForReview:   a.MarkedForReview,
					Revisited:         a.Revisited,
					ChangedAnswer:     a.ChangedAnswer,
					WasInitiallyWrong: a.WasInitiallyWrong,
				}
			}
			if err := attemptService.FinalizeSubmission(p.AssignmentID, p.AttemptID, p.StudentID, answers); err != nil {
				log.Printf("[QUEUE] finalize failed assignment %d attempt %d: %v", p.AssignmentID, p.AttemptID, err)
				return err
			}
			return nil
		},
	)

	// Auto-submit sweeper (Band C): finalize expired in-progress attempts.
	var sweeperCancel context.CancelFunc
	if cfg.RedisEnabled && redisClient != nil {
		sweeper := services.NewSweeper(attemptRepo, jobQueue, autosaveBuffer, cfg.SubmitGraceSeconds)
		sweeperCtx, cancel := context.WithCancel(context.Background())
		sweeperCancel = cancel
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-sweeperCtx.Done():
					return
				case <-ticker.C:
					sweeper.RunOnce(sweeperCtx)
				}
			}
		}()
	}

	// Shutdown hook drains buffered exam answers before the DB is closed so a
	// restart/deploy never loses the final ~1s of student input.
	shutdown := func() error {
		if sweeperCancel != nil {
			sweeperCancel()
		}
		jobQueue.Stop()
		if autosaveBuffer != nil {
			autosaveBuffer.Stop() // blocks until the final flush completes
		}
		return nil
	}

	return r, shutdown
}
