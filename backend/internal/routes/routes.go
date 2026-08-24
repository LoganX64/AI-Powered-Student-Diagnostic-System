package routes

import (
	"ai-student-diagnostic/backend/internal/auth"
	"ai-student-diagnostic/backend/internal/cache"
	"ai-student-diagnostic/backend/internal/config"
	handlers "ai-student-diagnostic/backend/internal/handler"
	"ai-student-diagnostic/backend/internal/liveview"
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
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Fatalf("invalid SetTrustedProxies(nil): %v", err)
		}
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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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
	notifRepo := repository.NewNotificationRepo(db)
	notifService := services.NewNotificationService(notifRepo, userRepo)
	studentRepo := repository.NewStudentRepo(db)
	coachRepo := repository.NewCoachRepo(db)
	testPaperRepo := repository.NewTestPaperRepo(db)
	assignmentRepo := repository.NewAssignmentRepo(db)
	attemptRepo := repository.NewAttemptRepo(db)
	loginAttemptRepo := repository.NewLoginAttemptRepo(db)
	batchRepo := repository.NewBatchRepo(db)
	jobRepo := repository.NewJobRepo(db)
	tenantRepo := repository.NewTenantRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	planRepo := repository.NewPlanRepo(db)
	subscriptionRepo := repository.NewSubscriptionRepo(db)

	// Quota middleware: caches subscription+usage per tenant.
	quotaMW := middleware.NewQuotaMiddleware(subscriptionRepo, planRepo)

	attemptService := services.NewAttemptService(attemptRepo, assignmentRepo, studentRepo, testPaperRepo)
	assignmentService := services.NewAssignmentService(assignmentRepo, studentRepo, testPaperRepo, coachRepo, userRepo, subscriptionRepo, cfg.RedisEnabled)
	jobService := services.NewJobService(jobRepo, attemptService, cfg.ComputeChunkSize, notifService)
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

	authHandler := auth.NewAuthHandler(authService, loginAttemptRepo, planRepo, subscriptionRepo, quotaMW)
	adminHandler := handlers.NewAdminHandler(userRepo, studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, jobRepo, attemptService, assignmentService, jobService, subscriptionRepo, jobQueue, cfg, quotaMW, notifService)
	coachHandler := handlers.NewCoachHandler(studentRepo, coachRepo, testPaperRepo, assignmentRepo, attemptRepo, batchRepo, jobRepo, attemptService, assignmentService, jobService, subscriptionRepo, jobQueue, cfg, quotaMW, notifService)
	studentHandler := handlers.NewStudentHandler(studentRepo, assignmentRepo, attemptRepo, testPaperRepo, attemptService, loginAttemptRepo, subscriptionRepo, jobQueue, autosaveBuffer, storageBackend, cfg, quotaMW, notifService)

	billingHandler := handlers.NewBillingHandler(planRepo, subscriptionRepo, studentRepo, coachRepo, storageBackend, quotaMW)

	liveviewHub := liveview.NewHub(redisClient)
	studentWSHandler := liveview.NewStudentWSHandler(liveviewHub, studentRepo, assignmentRepo)
	viewerWSHandler := liveview.NewViewerWSHandler(liveviewHub, studentRepo, assignmentRepo, coachRepo)
	videoHandler := handlers.NewVideoHandler(attemptRepo, assignmentRepo, studentRepo, coachRepo, storageBackend)
	studentHandler.VideoHandler = videoHandler

	superAdminHandler := handlers.NewSuperAdminHandler(tenantRepo, profileRepo, authService)
	profileHandler := handlers.NewProfileHandler(profileRepo, userRepo)
	tenantSettingsHandler := handlers.NewTenantSettingsHandler(tenantRepo)
	notifHandler := handlers.NewNotificationHandler(notifService, notifRepo)

	authRoute := r.Group("/auth")
	authRoute.Use(middleware.NewRateLimiter(redisClient, middleware.LoginLimit))
	{
		authRoute.POST("/login", authHandler.UserLogin)
		authRoute.POST("/register-admin", authHandler.RegisterAdmin)
	}

	student := r.Group("/student")
	{
		student.POST("/login", middleware.NewRateLimiter(redisClient, middleware.LoginLimit), studentHandler.StudentLogin)

		// Shared (Redis) rate limiter when available, else per-instance in-memory.
		limiter := middleware.NewRateLimiter(redisClient, middleware.DefaultLimit)

		protected := student.Group("")
		protected.Use(middleware.AuthMiddleware(studentRepo, userRepo))
		{
			protected.POST("/submit/:id", studentHandler.SubmitExam)
			protected.GET("/assignments", studentHandler.ListStudentAssignments)
			protected.GET("/assignments/:id/questions", studentHandler.GetAssignmentQuestions)
			protected.POST("/assignments/:id/start", studentHandler.StartExam)
			protected.POST("/assignments/:id/autosave", limiter, studentHandler.Autosave)
			protected.GET("/assignments/:id/state", studentHandler.GetState)
			protected.POST("/assignments/:id/submit", limiter, studentHandler.SubmitExam)

			protected.POST("/assignments/:id/video-chunk", limiter, studentHandler.VideoChunk)
			protected.GET("/assignments/:id/live", studentWSHandler.StudentLiveStream)
			protected.POST("/api/time", studentHandler.ServerTime)
		}
	}

	superAdmin := r.Group("/super-admin")
	superAdmin.Use(
		middleware.AuthMiddleware(studentRepo, userRepo),
		middleware.RoleMiddleware("super_admin"),
	)
	{
		superAdmin.GET("/stats", superAdminHandler.GetGlobalStats)
		superAdmin.GET("/tenants", superAdminHandler.ListTenants)
		superAdmin.POST("/tenants", superAdminHandler.CreateTenant)
		superAdmin.GET("/tenants/:id", superAdminHandler.GetTenant)
		superAdmin.PUT("/tenants/:id", superAdminHandler.UpdateTenant)
		superAdmin.PUT("/tenants/:id/suspend", superAdminHandler.SuspendTenant)
		superAdmin.PUT("/tenants/:id/reactivate", superAdminHandler.ReactivateTenant)
		superAdmin.GET("/tenants/:id/admins", superAdminHandler.ListTenantAdmins)
		superAdmin.POST("/tenants/:id/admins", superAdminHandler.CreateTenantAdmin)

		// Subscription plans (super admin only)
		superAdmin.GET("/plans", billingHandler.ListPlans)
		superAdmin.POST("/plans", billingHandler.CreatePlan)
		superAdmin.PUT("/plans/:id", billingHandler.UpdatePlan)
		superAdmin.DELETE("/plans/:id", billingHandler.DeletePlan)
		superAdmin.PUT("/tenants/:id/subscription", billingHandler.AssignPlan)
	}

	profile := r.Group("")
	profile.Use(middleware.AuthMiddleware(studentRepo, userRepo))
	{
		profile.GET("/auth/profile", profileHandler.GetProfile)
		profile.PUT("/auth/profile", profileHandler.UpdateProfile)
		profile.PUT("/auth/password", profileHandler.UpdatePassword)
	}

	admin := r.Group("/admin")
	admin.Use(
		middleware.AuthMiddleware(studentRepo, userRepo),
		middleware.RoleMiddleware("admin"),
	)
	{
		admin.POST("/register-coach", quotaMW.CheckCoachLimit(), authHandler.RegisterCoach)

		admin.POST("/subjects", adminHandler.CreateSubject)
		admin.POST("/students", quotaMW.CheckStudentLimit(), adminHandler.CreateStudent)
		admin.POST("/tests", quotaMW.CheckTestLimit(), adminHandler.CreateTest)
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
		admin.DELETE("/assignments/:id", adminHandler.DeleteAssignment)
		admin.DELETE("/students/:id", adminHandler.DeleteStudent)
		admin.PUT("/students/:id", adminHandler.UpdateStudent)
		admin.PUT("/students/:id/reactivate", adminHandler.ReactivateStudent)
		admin.GET("/coaches", adminHandler.ListCoaches)
		admin.GET("/coaches/:id", adminHandler.GetCoach)
		admin.DELETE("/coaches/:id", adminHandler.DeleteCoach)
		admin.PUT("/coaches/:id", adminHandler.UpdateCoach)
		admin.PUT("/coaches/:id/reactivate", adminHandler.ReactivateCoach)
		admin.GET("/coaches/:id/tests", adminHandler.ListCoachTests)
		admin.GET("/coaches/:id/students", adminHandler.ListCoachStudents)
		admin.GET("/subjects", adminHandler.ListSubjects)
		admin.DELETE("/subjects/:id", adminHandler.DeleteSubject)
		admin.PUT("/subjects/:id", adminHandler.UpdateSubject)
		admin.PUT("/subjects/:id/reactivate", adminHandler.ReactivateSubject)
		admin.GET("/assignments", adminHandler.ListAssignments)
		admin.GET("/students/:id/sqi", quotaMW.CheckSQIAccess(), adminHandler.GetStudentSQI)
		admin.POST("/students/sqi-batch", quotaMW.CheckSQIAccess(), adminHandler.GetStudentSQIBatch)
		admin.POST("/coaches/stats-batch", quotaMW.CheckSQIAccess(), adminHandler.GetCoachStatsBatch)

		admin.POST("/batches", adminHandler.CreateBatch)
		admin.GET("/batches", adminHandler.ListBatches)
		admin.DELETE("/batches/:id", adminHandler.DeleteBatch)
		admin.PUT("/batches/:id", adminHandler.UpdateBatch)
		admin.PATCH("/students/:id/batch", adminHandler.TransferStudentBatch)

		admin.POST("/sqi/compute", quotaMW.CheckSQIAccess(), adminHandler.ComputeSQI)
		admin.POST("/sqi/compute-batch", quotaMW.CheckSQIAccess(), adminHandler.ComputeSQIBatch)
		admin.GET("/jobs/:id", adminHandler.GetJob)

		admin.GET("/assignments/:id/video-chunks", quotaMW.CheckVideoProctoringAccess(), videoHandler.ListVideoChunks)
		admin.GET("/assignments/:id/video-chunk/:index", quotaMW.CheckVideoProctoringAccess(), videoHandler.StreamVideoChunk)
		admin.POST("/assignments/:id/video-token", quotaMW.CheckVideoProctoringAccess(), videoHandler.GenerateVideoToken)
		admin.DELETE("/assignments/:id/video", quotaMW.CheckVideoProctoringAccess(), videoHandler.DeleteVideo)

		admin.GET("/tenant/settings", tenantSettingsHandler.GetSettings)
		admin.PUT("/tenant/settings", tenantSettingsHandler.UpdateSettings)
		admin.PUT("/tenant", tenantSettingsHandler.UpdateTenantName)

		// Plans (admin can list plans for upgrade comparison)
		admin.GET("/plans", billingHandler.ListPlans)

		// Subscription (per-tenant)
		admin.GET("/subscription", billingHandler.GetSubscription)
		admin.POST("/subscription/checkout", billingHandler.CreateCheckout)
		admin.POST("/subscription/webhook", billingHandler.HandleWebhook)
		admin.POST("/subscription/cancel", billingHandler.CancelSubscription)

		admin.GET("/notifications", notifHandler.ListNotifications)
		admin.GET("/notifications/unread-count", notifHandler.UnreadCount)
		admin.PUT("/notifications/:id/read", notifHandler.MarkRead)
		admin.PUT("/notifications/read-all", notifHandler.MarkAllRead)
		admin.DELETE("/notifications/:id", notifHandler.DeleteNotification)
		admin.GET("/notifications/preferences", notifHandler.GetPreferences)
		admin.PUT("/notifications/preferences", notifHandler.UpdatePreferences)
	}

	videoStream := r.Group("/admin")
	videoStream.Use(middleware.VideoTokenMiddleware())
	{
		videoStream.GET("/assignments/:id/video-merged", quotaMW.CheckVideoProctoringAccess(), videoHandler.StreamMergedVideo)
	}

	view := r.Group("/view")
	view.Use(middleware.AuthMiddleware(studentRepo, userRepo))
	{
		view.GET("/students/:id/live", viewerWSHandler.ViewerLiveStream)
		view.GET("/students/:id/live/status", viewerWSHandler.LiveStatus)
	}

	coach := r.Group("/coach")
	coach.Use(
		middleware.AuthMiddleware(studentRepo, userRepo),
		middleware.RoleMiddleware("coach"),
	)
	{
		coach.GET("/students/:id/sqi", quotaMW.CheckSQIAccess(), coachHandler.GetStudentSQI)
		coach.POST("/students/sqi-batch", quotaMW.CheckSQIAccess(), adminHandler.GetStudentSQIBatch)

		coach.POST("/students", quotaMW.CheckStudentLimit(), coachHandler.CreateStudent)
		coach.POST("/tests", quotaMW.CheckTestLimit(), coachHandler.CreateTest)
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
		coach.DELETE("/assignments/:id", coachHandler.DeleteAssignment)
		coach.DELETE("/students/:id", adminHandler.DeleteStudent)
		coach.PUT("/students/:id", coachHandler.UpdateStudent)
		coach.PUT("/students/:id/reactivate", adminHandler.ReactivateStudent)
		coach.GET("/subjects", adminHandler.ListSubjects)
		coach.DELETE("/subjects/:id", adminHandler.DeleteSubject)
		coach.PUT("/subjects/:id", coachHandler.UpdateSubject)
		coach.PUT("/subjects/:id/reactivate", adminHandler.ReactivateSubject)
		coach.GET("/assignments", adminHandler.ListAssignments)
		coach.DELETE("/assignments/:id/video", videoHandler.DeleteVideo)

		coach.POST("/batches", coachHandler.CreateBatch)
		coach.GET("/batches", coachHandler.ListBatches)
		coach.DELETE("/batches/:id", coachHandler.DeleteBatch)
		coach.PUT("/batches/:id", coachHandler.UpdateBatch)
		coach.PATCH("/students/:id/batch", coachHandler.TransferStudentBatch)

		coach.POST("/sqi/compute", quotaMW.CheckSQIAccess(), coachHandler.ComputeSQI)
		coach.POST("/sqi/compute-batch", quotaMW.CheckSQIAccess(), coachHandler.ComputeSQIBatch)
		coach.GET("/jobs/:id", coachHandler.GetJob)

		coach.PUT("/password", authHandler.UpdatePassword)

		coach.GET("/notifications", notifHandler.ListNotifications)
		coach.GET("/notifications/unread-count", notifHandler.UnreadCount)
		coach.PUT("/notifications/:id/read", notifHandler.MarkRead)
		coach.PUT("/notifications/read-all", notifHandler.MarkAllRead)
		coach.DELETE("/notifications/:id", notifHandler.DeleteNotification)
		coach.GET("/notifications/preferences", notifHandler.GetPreferences)
		coach.PUT("/notifications/preferences", notifHandler.UpdatePreferences)
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
	// Runs in BOTH modes — the queue is mode-agnostic (it only calls
	// Queue.EnqueueFinalize, which works for the in-process and Redis queues),
	// so abandoned attempts are reclaimed even without Redis.
	var sweeperCancel context.CancelFunc
	sweeper := services.NewSweeper(attemptRepo, jobQueue, autosaveBuffer, cfg.SubmitGraceSeconds, notifService)
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
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[SWEEPER] panic during RunOnce: %v", r)
						}
					}()
					sweeper.RunOnce(sweeperCtx)
				}()
			}
		}
	}()

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
