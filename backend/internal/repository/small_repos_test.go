package repository

import (
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestJobCreateGetIncrement(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewJobRepo(db)
	tenantID := createTenant(t, db)

	id, err := r.Create(tenantID, "sqi_compute", []byte(`{"attempt_id":1}`), 10)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	j, err := r.Get(id, tenantID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if j.Type != "sqi_compute" || j.Total != 10 || j.Status != "pending" || j.Done != 0 {
		t.Fatalf("job=%+v", j)
	}

	if err := r.Increment(id, tenantID, 3, 1); err != nil {
		t.Fatalf("Increment: %v", err)
	}
	j, _ = r.Get(id, tenantID)
	if j.Done != 3 || j.Failed != 1 {
		t.Fatalf("after increment: %+v", j)
	}

	if err := r.SetStatus(id, tenantID, "completed"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	j, _ = r.Get(id, tenantID)
	if j.Status != "completed" {
		t.Fatalf("status=%q", j.Status)
	}
}

func TestJobGetNotFound(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewJobRepo(db)
	if _, err := r.Get(999999999, 1); err == nil {
		t.Fatal("expected error for missing job")
	}
}

func TestBatchCreateListExistsDelete(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewBatchRepo(db)
	tenantID := createTenant(t, db)

	id, err := r.Create(tenantID, "Batch A")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ok, _ := r.Exists(tenantID, id); !ok {
		t.Fatal("Exists must be true")
	}
	if ok, _ := r.Exists(tenantID, id+99999); ok {
		t.Fatal("Exists must be false for missing id")
	}

	list, err := r.List(tenantID)
	if err != nil || len(list) != 1 || list[0].Name != "Batch A" {
		t.Fatalf("List: %v %v", list, err)
	}

	ok, err := r.Update(tenantID, id, "Batch B")
	if err != nil || !ok {
		t.Fatalf("Update: %v %v", ok, err)
	}
	list, _ = r.List(tenantID)
	if list[0].Name != "Batch B" {
		t.Fatalf("after update: %+v", list[0])
	}

	reassigned, err := r.Delete(tenantID, id)
	if err != nil || reassigned != 0 {
		t.Fatalf("Delete: %d %v", reassigned, err)
	}
	if ok, _ := r.Exists(tenantID, id); ok {
		t.Fatal("batch must be deleted")
	}
}

func TestBatchSetStudentBatchAndMembers(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewBatchRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	batchID, _ := r.Create(tenantID, "Batch C")
	studentID := createStudent(t, db, tenantID, coachID, "batch-1", "Batch Student")

	if err := r.SetStudentBatch(tenantID, studentID, &batchID); err != nil {
		t.Fatalf("SetStudentBatch: %v", err)
	}
	members, err := r.MemberIDs(tenantID, batchID)
	if err != nil || len(members) != 1 || members[0] != studentID {
		t.Fatalf("MemberIDs: %v %v", members, err)
	}
	if err := r.SetStudentBatch(tenantID, studentID, nil); err != nil {
		t.Fatalf("SetStudentBatch nil: %v", err)
	}
	members, _ = r.MemberIDs(tenantID, batchID)
	if len(members) != 0 {
		t.Fatalf("after clearing: %v", members)
	}
}

func TestUserCreateAndGetters(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewUserRepo(db)
	tenantID := createTenant(t, db)
	email := uniqueEmail(t, "user")
	uid := createUser(t, db, tenantID, email, "admin")

	tid, err := r.GetTenantID(uid)
	if err != nil || tid != tenantID {
		t.Fatalf("GetTenantID=%d %v", tid, err)
	}
	if ok, _ := r.ExistsByID(uid); !ok {
		t.Fatal("ExistsByID must be true")
	}
	if ok, _ := r.ExistsByID(999999999); ok {
		t.Fatal("ExistsByID must be false")
	}
	hash, err := r.GetPasswordHash(uid)
	if err != nil || hash != "x" {
		t.Fatalf("GetPasswordHash=%q %v", hash, err)
	}
	if err := r.UpdatePassword(uid, "new-hash"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	hash, _ = r.GetPasswordHash(uid)
	if hash != "new-hash" {
		t.Fatalf("hash=%q", hash)
	}
}

func TestUserLoginLookup(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewUserRepo(db)
	tenantID := createTenant(t, db)
	email := uniqueEmail(t, "login")
	createUser(t, db, tenantID, email, "admin")

	row, err := r.GetByEmailWithCoachCheck(email)
	if err != nil {
		t.Fatalf("GetByEmailWithCoachCheck: %v", err)
	}
	if row.UserID == 0 || row.Role != "admin" || !row.TenantID.Valid {
		t.Fatalf("row=%+v", row)
	}
	if _, err := r.GetByEmailWithCoachCheck("no-such-" + email); err == nil {
		t.Fatal("expected error for unknown email")
	}
}

func TestProfileGetAndUpdate(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewProfileRepo(db)
	tenantID := createTenant(t, db)
	email := uniqueEmail(t, "prof")
	uid := createUser(t, db, tenantID, email, "admin")

	// No profile row yet: GetByUserID must still succeed with NULLs.
	p, err := r.GetByUserID(uid)
	if err != nil {
		t.Fatalf("GetByUserID (no profile): %v", err)
	}
	if p.Email != email || p.TenantName == nil || *p.TenantName == "" {
		t.Fatalf("p=%+v", p)
	}

	if err := r.UpdateProfile(uid, "Fixture Name", "9999"); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	p, err = r.GetByUserID(uid)
	if err != nil {
		t.Fatalf("GetByUserID after update: %v", err)
	}
	if p.DisplayName == nil || *p.DisplayName != "Fixture Name" || p.Phone == nil || *p.Phone != "9999" {
		t.Fatalf("p=%+v", p)
	}

	if err := r.UpdateEmail(uid, "new-"+email); err != nil {
		t.Fatalf("UpdateEmail: %v", err)
	}
	p, _ = r.GetByUserID(uid)
	if p.Email != "new-"+email {
		t.Fatalf("email=%q", p.Email)
	}
}

func TestCoachGetDetailAndSubjects(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewCoachRepo(db)
	tenantID := createTenant(t, db)
	coachUserID, coachID := createCoach(t, db, tenantID)

	if ok, _ := r.Exists(coachID, tenantID); !ok {
		t.Fatal("Exists must be true")
	}
	if ok, _ := r.Exists(coachID, tenantID+99999); ok {
		t.Fatal("Exists must be false for other tenant")
	}
	gotID, err := r.GetIDFromUser(coachUserID)
	if err != nil || gotID != coachID {
		t.Fatalf("GetIDFromUser=%d %v", gotID, err)
	}
	gotID, gotTenant, err := r.GetIDAndTenantFromUser(coachUserID)
	if err != nil || gotID != coachID || gotTenant != tenantID {
		t.Fatalf("GetIDAndTenantFromUser=(%d,%d) %v", gotID, gotTenant, err)
	}

	detail, err := r.GetDetail(coachID, tenantID)
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if detail == nil {
		t.Fatal("GetDetail returned nil")
	}

	subs, err := r.GetCoachSubjects(coachID)
	if err != nil {
		t.Fatalf("GetCoachSubjects: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("subs=%v", subs)
	}
}

func TestTenantGetByIDAndUpdate(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTenantRepo(db)
	tenantID := createTenant(t, db)

	tenant, err := r.GetByID(tenantID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if tenant.ID != tenantID || tenant.Name == "" {
		t.Fatalf("tenant=%+v", tenant)
	}

	if err := r.Update(tenantID, "Renamed Tenant"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	tenant, _ = r.GetByID(tenantID)
	if tenant.Name != "Renamed Tenant" {
		t.Fatalf("name=%q", tenant.Name)
	}

	if err := r.Suspend(tenantID); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	tenant, _ = r.GetByID(tenantID)
	if tenant.SuspendedAt == nil {
		t.Fatal("SuspendedAt must be set after Suspend")
	}
	if err := r.Reactivate(tenantID); err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	tenant, _ = r.GetByID(tenantID)
	if tenant.SuspendedAt != nil {
		t.Fatal("SuspendedAt must be nil after Reactivate")
	}
}

func TestTenantGetNotFound(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTenantRepo(db)
	// Contract: a missing tenant returns (nil, nil); the caller checks for a nil
	// row and turns it into a 404 rather than relying on an error.
	tenant, err := r.GetByID(999999999)
	if err != nil {
		t.Fatalf("GetByID missing: %v", err)
	}
	if tenant != nil {
		t.Fatalf("expected nil row for missing tenant, got %+v", tenant)
	}
}

func TestStudentGettersAndList(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewStudentRepo(db)
	tenantID := createTenant(t, db)
	_, coachID := createCoach(t, db, tenantID)
	studentID := createStudent(t, db, tenantID, coachID, "st-1", "Student One")

	if ok, _ := r.Exists(studentID, tenantID); !ok {
		t.Fatal("Exists must be true")
	}
	if ok, _ := r.ExistsActive(studentID, tenantID, coachID); !ok {
		t.Fatal("ExistsActive must be true")
	}
	name, err := r.GetName(studentID, tenantID)
	if err != nil || name != "Student One" {
		t.Fatalf("GetName=%q %v", name, err)
	}
	gotName, code, err := r.GetNameCode(studentID, tenantID)
	if err != nil || gotName != "Student One" || code != "st-1" {
		t.Fatalf("GetNameCode=(%q,%q) %v", gotName, code, err)
	}
	sid, gotTenant, err := r.GetIDByStudentCode("st-1")
	if err != nil || sid != studentID || gotTenant != tenantID {
		t.Fatalf("GetIDByStudentCode=(%d,%d) %v", sid, gotTenant, err)
	}

	list, total, err := r.List(tenantID, nil, false, "", nil, 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List: %v %d %v", err, total, list)
	}
}

func TestStudentSoftDeleteReactivate(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewStudentRepo(db)
	tenantID := createTenant(t, db)
	adminID := createUser(t, db, tenantID, uniqueEmail(t, "admin"), "admin")
	_, coachID := createCoach(t, db, tenantID)
	studentID := createStudent(t, db, tenantID, coachID, "st-2", "Student Two")

	ok, err := r.SoftDelete(studentID, tenantID, adminID, nil)
	if err != nil || !ok {
		t.Fatalf("SoftDelete: %v %v", ok, err)
	}
	ok, err = r.ExistsActive(studentID, tenantID, coachID)
	if err != nil || ok {
		t.Fatalf("ExistsActive after delete: %v %v", ok, err)
	}
	ok, err = r.Reactivate(studentID, tenantID, nil)
	if err != nil || !ok {
		t.Fatalf("Reactivate: %v %v", ok, err)
	}
	ok, err = r.ExistsActive(studentID, tenantID, coachID)
	if err != nil || !ok {
		t.Fatalf("ExistsActive after reactivate: %v %v", ok, err)
	}
}
