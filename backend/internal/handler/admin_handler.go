package handlers

import (
	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
)

type AdminHandler struct {
	UserRepo          *repository.UserRepo
	StudentRepo       *repository.StudentRepo
	CoachRepo         *repository.CoachRepo
	TestPaperRepo     *repository.TestPaperRepo
	AssignmentRepo    *repository.AssignmentRepo
	AttemptRepo       *repository.AttemptRepo
	BatchRepo         *repository.BatchRepo
	JobRepo           *repository.JobRepo
	AttemptService    *services.AttemptService
	AssignmentService *services.AssignmentService
	JobService        *services.JobService
	SubscriptionRepo  *repository.SubscriptionRepo
	Queue             queue.Queue
	Cfg               *config.Config
	// QuotaMW is optional (nil in tests). Guarded before every use.
	QuotaMW *middleware.QuotaMiddleware
}

func NewAdminHandler(
	userRepo *repository.UserRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testPaperRepo *repository.TestPaperRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	batchRepo *repository.BatchRepo,
	jobRepo *repository.JobRepo,
	attemptService *services.AttemptService,
	assignmentService *services.AssignmentService,
	jobService *services.JobService,
	subscriptionRepo *repository.SubscriptionRepo,
	q queue.Queue,
	cfg *config.Config,
	quotaMW *middleware.QuotaMiddleware,
) *AdminHandler {
	return &AdminHandler{
		UserRepo:          userRepo,
		StudentRepo:       studentRepo,
		CoachRepo:         coachRepo,
		TestPaperRepo:     testPaperRepo,
		AssignmentRepo:    assignmentRepo,
		AttemptRepo:       attemptRepo,
		BatchRepo:         batchRepo,
		JobRepo:           jobRepo,
		AttemptService:    attemptService,
		AssignmentService: assignmentService,
		JobService:        jobService,
		SubscriptionRepo:  subscriptionRepo,
		Queue:             q,
		Cfg:               cfg,
		QuotaMW:           quotaMW,
	}
}
