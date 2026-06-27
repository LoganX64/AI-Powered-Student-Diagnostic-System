package handlers

import (
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
	AttemptService    *services.AttemptService
	AssignmentService *services.AssignmentService
}

func NewAdminHandler(
	userRepo *repository.UserRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testPaperRepo *repository.TestPaperRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	attemptService *services.AttemptService,
	assignmentService *services.AssignmentService,
) *AdminHandler {
	return &AdminHandler{
		UserRepo:          userRepo,
		StudentRepo:       studentRepo,
		CoachRepo:         coachRepo,
		TestPaperRepo:     testPaperRepo,
		AssignmentRepo:    assignmentRepo,
		AttemptRepo:       attemptRepo,
		AttemptService:    attemptService,
		AssignmentService: assignmentService,
	}
}
