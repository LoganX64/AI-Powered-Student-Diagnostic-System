package services

import (
	"database/sql"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
	"ai-student-diagnostic/backend/utils"
)

func newAuthService(t *testing.T, db *sql.DB) *AuthService {
	t.Helper()
	return NewAuthService(repository.NewUserRepo(db), repository.NewCoachRepo(db))
}

func TestAuthServiceUserLogin(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)
	email := testutil.UniqueEmail(t, "authlogin")
	hashed, err := utils.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	uid := testutil.CreateUser(t, db, tenantID, email, "admin")
	// CreateUser stores a placeholder password; replace it with a real bcrypt hash.
	if _, err := db.Exec(`UPDATE users SET password = $1 WHERE id = $2`, hashed, uid); err != nil {
		t.Fatalf("set password: %v", err)
	}

	svc := newAuthService(t, db)
	res, err := svc.UserLogin(email, "correct-horse")
	if err != nil {
		t.Fatalf("UserLogin: %v", err)
	}
	if res.UserID != uid || res.Role != "admin" || int(res.TenantID) != tenantID {
		t.Fatalf("result=%+v", res)
	}
}

func TestAuthServiceUserLoginWrongPassword(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)
	email := testutil.UniqueEmail(t, "authwrong")
	hashed, _ := utils.HashPassword("correct-horse")
	uid := testutil.CreateUser(t, db, tenantID, email, "admin")
	if _, err := db.Exec(`UPDATE users SET password = $1 WHERE id = $2`, hashed, uid); err != nil {
		t.Fatalf("set password: %v", err)
	}

	svc := newAuthService(t, db)
	if _, err := svc.UserLogin(email, "battery-staple"); err == nil {
		t.Fatal("wrong password must be rejected")
	}
}

func TestAuthServiceUserLoginUnknownEmail(t *testing.T) {
	db := testutil.OpenTestDB(t)
	svc := newAuthService(t, db)
	if _, err := svc.UserLogin("nobody@example.test", "x"); err == nil {
		t.Fatal("unknown email must be rejected")
	}
}

func TestAuthServiceRegisterAdminCreatesTenantAndUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	svc := newAuthService(t, db)
	email := testutil.UniqueEmail(t, "regadmin")

	tenantID, userID, err := svc.RegisterAdmin(email, "hashed", "New Org")
	if err != nil {
		t.Fatalf("RegisterAdmin: %v", err)
	}
	// Clean up the tenant this created (RegisterAdmin has no fixture cleanup).
	t.Cleanup(func() { db.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID) })

	if tenantID == 0 || userID == 0 {
		t.Fatalf("tenant=%d user=%d", tenantID, userID)
	}
	// The admin must be resolvable by email and report role admin.
	row, err := repository.NewUserRepo(db).GetByEmailWithCoachCheck(email)
	if err != nil {
		t.Fatalf("GetByEmailWithCoachCheck: %v", err)
	}
	if row.Role != "admin" || !row.TenantID.Valid || int(row.TenantID.Int32) != tenantID {
		t.Fatalf("row=%+v want tenant %d admin", row, tenantID)
	}
}

func TestAuthServiceCreateAdminForTenant(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)
	svc := newAuthService(t, db)
	email := testutil.UniqueEmail(t, "cadmin")

	userID, err := svc.CreateAdminForTenant(tenantID, email, "hashed", "Admin Name")
	if err != nil {
		t.Fatalf("CreateAdminForTenant: %v", err)
	}
	if userID == 0 {
		t.Fatal("no user id returned")
	}
}

func TestAuthServiceCreateAdminDuplicateEmail(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)
	email := testutil.UniqueEmail(t, "dupadmin")
	testutil.CreateUser(t, db, tenantID, email, "admin")

	svc := newAuthService(t, db)
	if _, err := svc.CreateAdminForTenant(tenantID, email, "hashed", "Name"); err == nil {
		t.Fatal("duplicate email must be rejected")
	}
}

func TestAuthServiceRegisterCoach(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tenantID := testutil.CreateTenant(t, db)
	adminUserID := testutil.CreateAdmin(t, db, tenantID)
	subjectID := testutil.CreateSubject(t, db, tenantID, "Physics")

	svc := newAuthService(t, db)
	email := testutil.UniqueEmail(t, "coachreg")
	userID, coachID, err := svc.RegisterCoach(adminUserID, email, "hashed", "New Coach", []int{subjectID})
	if err != nil {
		t.Fatalf("RegisterCoach: %v", err)
	}
	if userID == 0 || coachID == 0 {
		t.Fatalf("user=%d coach=%d", userID, coachID)
	}
	// The coach and its subject link must exist.
	subs, err := repository.NewCoachRepo(db).GetCoachSubjects(coachID)
	if err != nil {
		t.Fatalf("GetCoachSubjects: %v", err)
	}
	if len(subs) != 1 || subs[0].SubjectID != subjectID {
		t.Fatalf("subjects=%+v", subs)
	}
}
