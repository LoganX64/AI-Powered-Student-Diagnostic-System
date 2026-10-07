package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

// handler_fixtures_test.go builds the three handler structs that own almost
// every handler method: AdminHandler, CoachHandler and StudentHandler. Each
// fixture wires the real repositories over a throwaway tenant graph and supplies
// offline stand-ins for the three infrastructure dependencies:
//
//	queue   -> in-process implementation (Redis disabled)
//	storage -> local filesystem under t.TempDir()
//	autosave-> nil-redis buffer, which is a documented no-op
//
// Tenant scoping is the point of most assertions here, so every fixture also
// carries a second tenant that belongs to nobody in the request context.

// handlers bundles the three handlers plus the repos and ids tests need.
type handlerFixture struct {
	DB *sql.DB

	Admin   *AdminHandler
	Coach   *CoachHandler
	Student *StudentHandler

	// Repos, for the few tests that assert persistence directly.
	UserRepo         *repository.UserRepo
	StudentRepo      *repository.StudentRepo
	CoachRepo        *repository.CoachRepo
	TestPaperRepo    *repository.TestPaperRepo
	AssignmentRepo   *repository.AssignmentRepo
	AttemptRepo      *repository.AttemptRepo
	NotificationRepo *repository.NotificationRepo
	SubscriptionRepo *repository.SubscriptionRepo
	ProfileRepo      *repository.ProfileRepo
	TenantRepo       *repository.TenantRepo

	Storage storage.Storage

	// Primary tenant graph.
	TenantID     int
	AdminUserID  int
	CoachUserID  int
	CoachID      int
	SubjectID    int
	TestID       int
	StudentID    int
	AssignmentID int

	// OtherTenantID belongs to a separate tenant. Any handler reached with
	// OtherTenantID's ids while the context says TenantID must refuse.
	OtherTenantID     int
	OtherCoachUserID  int
	OtherCoachID      int
	OtherSubjectID    int
	OtherTestID       int
	OtherStudentID    int
	OtherAssignmentID int
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.OpenTestDB(t)

	f := &handlerFixture{DB: db}

	f.UserRepo = repository.NewUserRepo(db)
	f.StudentRepo = repository.NewStudentRepo(db)
	f.CoachRepo = repository.NewCoachRepo(db)
	f.TestPaperRepo = repository.NewTestPaperRepo(db)
	f.AssignmentRepo = repository.NewAssignmentRepo(db)
	f.AttemptRepo = repository.NewAttemptRepo(db)
	f.NotificationRepo = repository.NewNotificationRepo(db)
	f.SubscriptionRepo = repository.NewSubscriptionRepo(db)
	f.ProfileRepo = repository.NewProfileRepo(db)
	f.TenantRepo = repository.NewTenantRepo(db)
	batchRepo := repository.NewBatchRepo(db)
	jobRepo := repository.NewJobRepo(db)
	loginAttemptRepo := repository.NewLoginAttemptRepo(db)

	// Infrastructure stand-ins: no Redis, no object store.
	q := queue.New(&config.Config{})
	st := storage.NewLocal(t.TempDir())
	autosave := services.NewAutosaveBuffer(nil, f.AttemptRepo)
	cfg := &config.Config{ScaleBandC: 50000, SubmitGraceSeconds: 30, UploadDir: t.TempDir()}
	f.Storage = st

	attemptService := services.NewAttemptService(f.AttemptRepo, f.AssignmentRepo, f.StudentRepo, f.TestPaperRepo)
	assignmentService := services.NewAssignmentService(f.AssignmentRepo, f.StudentRepo, f.TestPaperRepo, f.CoachRepo, f.UserRepo, f.SubscriptionRepo, false)
	notifService := services.NewNotificationService(f.NotificationRepo, f.UserRepo)
	jobService := services.NewJobService(jobRepo, attemptService, 100, notifService)
	// QuotaMW stays nil here; quota behaviour is a middleware concern and the
	// handlers document the field as optional.
	var quotaMW *middleware.QuotaMiddleware

	f.Admin = NewAdminHandler(f.UserRepo, f.StudentRepo, f.CoachRepo, f.TestPaperRepo,
		f.AssignmentRepo, f.AttemptRepo, batchRepo, jobRepo,
		attemptService, assignmentService, jobService, f.SubscriptionRepo,
		q, cfg, quotaMW, notifService)

	f.Coach = NewCoachHandler(f.StudentRepo, f.CoachRepo, f.TestPaperRepo, f.AssignmentRepo,
		f.AttemptRepo, batchRepo, jobRepo,
		attemptService, assignmentService, jobService, f.SubscriptionRepo,
		q, cfg, quotaMW, notifService)

	f.Student = NewStudentHandler(f.StudentRepo, f.AssignmentRepo, f.AttemptRepo, f.TestPaperRepo,
		attemptService, loginAttemptRepo, f.SubscriptionRepo,
		q, autosave, st, cfg, quotaMW, notifService)

	// Primary tenant.
	f.TenantID = testutil.CreateTenant(t, db)
	f.AdminUserID = testutil.CreateAdmin(t, db, f.TenantID)
	f.CoachUserID, f.CoachID = testutil.CreateCoach(t, db, f.TenantID)
	f.SubjectID = testutil.CreateSubject(t, db, f.TenantID, "Algebra")
	f.TestID = testutil.CreateTestNamed(t, db, f.TenantID, f.SubjectID, f.CoachID, 60, "Algebra Paper", "Algebra")
	testutil.CreateQuestion(t, db, f.TestID)
	f.StudentID = testutil.CreateStudent(t, db, f.TenantID, f.CoachID, testutil.UniqueCode(t, "stu"), "Primary Student")
	f.AssignmentID = testutil.CreateAssignment(t, db, f.StudentID, f.TestID, f.CoachID)

	// Foreign tenant, used to prove scoping.
	f.OtherTenantID = testutil.CreateTenant(t, db)
	f.OtherCoachUserID, f.OtherCoachID = testutil.CreateCoach(t, db, f.OtherTenantID)
	f.OtherSubjectID = testutil.CreateSubject(t, db, f.OtherTenantID, "Foreign Subject")
	f.OtherTestID = testutil.CreateTestNamed(t, db, f.OtherTenantID, f.OtherSubjectID, f.OtherCoachID, 45, "Foreign Paper", "Foreign Subject")
	testutil.CreateQuestion(t, db, f.OtherTestID)
	f.OtherStudentID = testutil.CreateStudent(t, db, f.OtherTenantID, f.OtherCoachID, testutil.UniqueCode(t, "other"), "Foreign Student")
	f.OtherAssignmentID = testutil.CreateAssignment(t, db, f.OtherStudentID, f.OtherTestID, f.OtherCoachID)

	return f
}

// ctx builds a gin context pre-seeded with the identity a handler expects.
func (f *handlerFixture) ctx(t *testing.T, role string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Set("role", role)
	return c, w
}

// ctxAsStudent seeds the context with a student's identity instead of a user's.
func (f *handlerFixture) ctxAsStudent(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("student_id", f.StudentID)
	c.Set("role", "student")
	return c, w
}

// ctxAsCoach seeds the context with the coach's user identity.
func (f *handlerFixture) ctxAsCoach(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.CoachUserID)
	c.Set("role", "coach")
	return c, w
}

// withParam sets a single path param (e.g. "id") on the context.
func withParam(c *gin.Context, name, value string) {
	c.Params = gin.Params{{Key: name, Value: value}}
}
