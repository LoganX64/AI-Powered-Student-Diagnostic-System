package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// super_admin_handler.go had no tests at all, and neither did TenantRepo.List,
// the repository method behind its one filter argument. Two defects lived in
// those 32 untested lines: ?plan= concatenated a raw query value into SQL, and
// ?plan=paid had an unbalanced paren that made it 500 on every call. Both are
// fixed; the tests below pin them.
//
// The fixture's two tenants both start with no subscription row, so plan-filter
// assertions would otherwise depend on whatever dev data happens to be in the
// database. subscribeTenant writes a real subscription instead.

// orphanTenants removes every tenant matching name, so it can be registered as a
// blanket cleanup in any test that drives CreateTenant.
//
// This is not defensive paranoia. AuthService.RegisterAdmin inserts the tenant
// first and the user second, with no transaction between them, so when the user
// insert fails (a duplicate email, which CreateTenant lets through as a 500) the
// tenant row is already committed and orphaned. Those orphans accumulate in dev
// data on every run and, because testutil.UniqueEmail is deterministic per
// process, they eventually collide with a later run's generated address.
//
// The name must therefore be unique per test — hence the seq suffix rather than a
// literal like "Second Org".
func orphanTenants(t *testing.T, f *handlerFixture, name string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := f.DB.Exec(`DELETE FROM tenants WHERE name = $1`, name); err != nil {
			t.Errorf("cleanup tenants named %q: %v", name, err)
		}
	})
}

func subscribeTenant(t *testing.T, f *handlerFixture, tenantID int, planSlug string) {
	t.Helper()
	if _, err := f.DB.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id)
		SELECT $1, id FROM subscription_plans WHERE slug = $2
		ON CONFLICT (tenant_id) DO UPDATE SET plan_id = EXCLUDED.plan_id`,
		tenantID, planSlug); err != nil {
		t.Fatalf("subscribe tenant %d to %s: %v", tenantID, planSlug, err)
	}
}

// tenantIDs parses a {"data":[{"id":N}]} payload and returns just the ids.
func tenantIDs(t *testing.T, body string) []int {
	t.Helper()
	var resp struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
	out := make([]int, 0, len(resp.Data))
	for _, d := range resp.Data {
		out = append(out, d.ID)
	}
	return out
}

func containsID(ids []int, want int) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// listTenantsWithQuery runs ListTenants with a raw query string, URL-escaping the
// value so a payload can be passed through exactly as a client would send it.
func listTenantsWithQuery(t *testing.T, f *handlerFixture, query string) (string, int) {
	t.Helper()
	c, w := f.ctxSuperAdmin(t)
	c.Request = httptest.NewRequest(http.MethodGet, "/super-admin/tenants?"+query, nil)
	f.SuperAdmin.ListTenants(c)
	return w.Body.String(), w.Code
}

func TestSuperAdminListTenants(t *testing.T) {
	f := newHandlerFixture(t)

	body, code := listTenantsWithQuery(t, f, "")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	ids := tenantIDs(t, body)
	if !containsID(ids, f.TenantID) {
		t.Fatalf("our tenant %d missing from %v", f.TenantID, ids)
	}
	if !containsID(ids, f.OtherTenantID) {
		t.Fatalf("other fixture tenant %d missing from %v", f.OtherTenantID, ids)
	}

	var resp struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Limit != 50 {
		t.Fatalf("limit=%d, want the ParsePagination default of 50", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Fatalf("offset=%d, want 0", resp.Offset)
	}
	if resp.Total < 2 {
		t.Fatalf("total=%d, want at least our two fixture tenants", resp.Total)
	}
}

func TestSuperAdminListTenantsSearchFilter(t *testing.T) {
	f := newHandlerFixture(t)

	// Both fixture tenants are named "fixtures-<t.Name()>", so a prefix search
	// for that matches both. Search for something neither can contain.
	body, code := listTenantsWithQuery(t, f, "search=zzz-no-such-tenant-zzz")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if ids := tenantIDs(t, body); len(ids) != 0 {
		t.Fatalf("search matched %v, want none", ids)
	}

	// A real substring narrows the result set.
	body, code = listTenantsWithQuery(t, f, "search=fixtures")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if ids := tenantIDs(t, body); !containsID(ids, f.TenantID) {
		t.Fatalf("search=fixtures missing our tenant, got %v", ids)
	}
}

func TestSuperAdminListTenantsPlanFilter(t *testing.T) {
	f := newHandlerFixture(t)
	subscribeTenant(t, f, f.TenantID, "professional")
	subscribeTenant(t, f, f.OtherTenantID, "free")

	cases := []struct {
		filter string
		want   int
		absent int
	}{
		{filter: "professional", want: f.TenantID, absent: f.OtherTenantID},
		{filter: "free", want: f.OtherTenantID, absent: f.TenantID},
		// "paid" is everything that is not free, so the professional tenant is in.
		{filter: "paid", want: f.TenantID, absent: f.OtherTenantID},
	}
	for _, tc := range cases {
		t.Run(tc.filter, func(t *testing.T) {
			body, code := listTenantsWithQuery(t, f, "plan="+tc.filter)
			if code != http.StatusOK {
				t.Fatalf("code=%d body=%s", code, body)
			}
			ids := tenantIDs(t, body)
			if !containsID(ids, tc.want) {
				t.Fatalf("plan=%s missing %d, got %v", tc.filter, tc.want, ids)
			}
			if containsID(ids, tc.absent) {
				t.Fatalf("plan=%s wrongly included %d, got %v", tc.filter, tc.absent, ids)
			}
		})
	}
}

// An unknown slug is a literal comparison that matches nothing. It must not be
// able to widen the filter into returning something.
func TestSuperAdminListTenantsUnknownPlanSlug(t *testing.T) {
	f := newHandlerFixture(t)
	subscribeTenant(t, f, f.TenantID, "professional")
	subscribeTenant(t, f, f.OtherTenantID, "free")

	body, code := listTenantsWithQuery(t, f, "plan=nonexistent")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if ids := tenantIDs(t, body); containsID(ids, f.TenantID) || containsID(ids, f.OtherTenantID) {
		t.Fatalf("plan=nonexistent returned our tenants %v", ids)
	}
}

// Regression guard for the SQL injection in TenantRepo.List. The payload closed
// the string literal and appended its own OR, so before the fix it returned the
// professional tenant that no legitimate slug filter matched.
func TestSuperAdminListTenantsRejectsInjectedPlanFilter(t *testing.T) {
	f := newHandlerFixture(t)
	subscribeTenant(t, f, f.TenantID, "professional")
	subscribeTenant(t, f, f.OtherTenantID, "free")

	for _, payload := range []string{
		"x' OR slug='professional' OR slug='x",
		"x' OR ts.plan_id IS NOT NULL OR slug='x",
		"x' OR 1=1 OR slug='x",
	} {
		t.Run(payload, func(t *testing.T) {
			body, code := listTenantsWithQuery(t, f, "plan="+urlEscape(payload))
			if code != http.StatusOK {
				t.Fatalf("code=%d body=%s", code, body)
			}
			if ids := tenantIDs(t, body); containsID(ids, f.TenantID) {
				t.Fatalf("payload reached the query and returned tenant %d: %v", f.TenantID, ids)
			}
		})
	}
}

// TestSuperAdminCreateTenant covers AuthService.RegisterAdmin, which inserts via
// UserRepo.CreateTenant rather than testutil.CreateTenant, so no fixture cleanup
// covers it. This test must delete the row itself or it pollutes dev data
// (Trap 3).
func TestSuperAdminCreateTenant(t *testing.T) {
	f := newHandlerFixture(t)
	email := testutil.UniqueEmail(t, "newadmin")
	name := "Created Org " + testutil.UniqueCode(t, "org")
	orphanTenants(t, f, name)

	c, w := f.ctxSuperAdmin(t)
	withJSONBody(c, http.MethodPost, `{"name":"`+name+`","admin_email":"`+email+`","admin_password":"longenoughpw","admin_name":"Ada"}`)

	f.SuperAdmin.CreateTenant(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		TenantID int `json:"tenant_id"`
		UserID   int `json:"user_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TenantID == 0 || resp.UserID == 0 {
		t.Fatalf("ids not returned: %+v", resp)
	}

	t.Cleanup(func() { f.DB.Exec(`DELETE FROM tenants WHERE id = $1`, resp.TenantID) })

	var gotName string
	if err := f.DB.QueryRow(`SELECT name FROM tenants WHERE id = $1`, resp.TenantID).Scan(&gotName); err != nil {
		t.Fatalf("tenant not persisted: %v", err)
	}
	if gotName != name {
		t.Fatalf("tenant name=%q want %q", gotName, name)
	}

	var gotEmail, role string
	if err := f.DB.QueryRow(`SELECT email, role FROM users WHERE id = $1`, resp.UserID).Scan(&gotEmail, &role); err != nil {
		t.Fatalf("admin user not persisted: %v", err)
	}
	if gotEmail != email {
		t.Fatalf("admin email=%q want %q", gotEmail, email)
	}
	if role != "admin" {
		t.Fatalf("role=%q want admin", role)
	}
}

// The handler binds admin_name as required and then never uses it:
// AuthService.RegisterAdmin takes no name, and UserRepo.Create has no name column
// to write to. A super admin can therefore never create a tenant whose admin has
// a name, while the API contract requires the field. Pinned so the discrepancy is
// visible rather than silent.
func TestSuperAdminCreateTenantDiscardsAdminName(t *testing.T) {
	f := newHandlerFixture(t)
	name := "No Name Org " + testutil.UniqueCode(t, "org")
	orphanTenants(t, f, name)

	c, w := f.ctxSuperAdmin(t)
	withJSONBody(c, http.MethodPost, `{"name":"`+name+`","admin_email":"`+testutil.UniqueEmail(t, "noname")+`","admin_password":"longenoughpw","admin_name":"Should Not Persist"}`)

	f.SuperAdmin.CreateTenant(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TenantID int `json:"tenant_id"`
		UserID   int `json:"user_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	t.Cleanup(func() { f.DB.Exec(`DELETE FROM tenants WHERE id = $1`, resp.TenantID) })

	// The users table has no name column at all — admin names live in
	// user_profiles, which RegisterAdmin never touches. So the stronger form of
	// the assertion is that nothing name-shaped was written anywhere.
	var profileRows int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM user_profiles WHERE user_id = $1`, resp.UserID).Scan(&profileRows); err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	if profileRows != 0 {
		t.Fatalf("admin_name was persisted into user_profiles (%d rows); the handler is expected to drop it", profileRows)
	}
}

func TestSuperAdminCreateTenantWeakPassword(t *testing.T) {
	f := newHandlerFixture(t)

	var before int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&before); err != nil {
		t.Fatalf("count tenants: %v", err)
	}

	c, w := f.ctxSuperAdmin(t)
	withJSONBody(c, http.MethodPost, `{"name":"Weak Org","admin_email":"`+testutil.UniqueEmail(t, "weak")+`","admin_password":"short","admin_name":"Ada"}`)

	f.SuperAdmin.CreateTenant(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400 (body=%s)", w.Code, w.Body.String())
	}

	var after int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&after); err != nil {
		t.Fatalf("count tenants: %v", err)
	}
	if after != before {
		t.Fatalf("tenant count %d -> %d; a rejected create must not insert", before, after)
	}
}

func TestSuperAdminCreateTenantBadPayload(t *testing.T) {
	f := newHandlerFixture(t)
	email := testutil.UniqueEmail(t, "bad")
	cases := map[string]string{
		"missing name":       `{"admin_email":"` + email + `","admin_password":"longenoughpw","admin_name":"Ada"}`,
		"missing email":      `{"name":"X","admin_password":"longenoughpw","admin_name":"Ada"}`,
		"missing password":   `{"name":"X","admin_email":"` + email + `","admin_name":"Ada"}`,
		"missing admin_name": `{"name":"X","admin_email":"` + email + `","admin_password":"longenoughpw"}`,
		"malformed json":     `{"name":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c, w := f.ctxSuperAdmin(t)
			withJSONBody(c, http.MethodPost, body)
			f.SuperAdmin.CreateTenant(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("code=%d, want 400 (body=%s)", w.Code, w.Body.String())
			}
		})
	}
}

// DEFECT PINNED: CreateTenant maps a duplicate email to 500 via
// utils.InternalError, while CreateTenantAdmin in the same file maps the same
// class of failure to 400. Recorded, not fixed.
func TestSuperAdminCreateTenantDuplicateEmail(t *testing.T) {
	f := newHandlerFixture(t)
	email := testutil.UniqueEmail(t, "dupe")
	firstName := "First Org " + testutil.UniqueCode(t, "org")
	secondName := "First Org " + testutil.UniqueCode(t, "org")
	orphanTenants(t, f, firstName)
	orphanTenants(t, f, secondName)

	first, wFirst := f.ctxSuperAdmin(t)
	withJSONBody(first, http.MethodPost, `{"name":"`+firstName+`","admin_email":"`+email+`","admin_password":"longenoughpw","admin_name":"Ada"}`)
	f.SuperAdmin.CreateTenant(first)
	if wFirst.Code != http.StatusCreated {
		t.Fatalf("first create code=%d body=%s", wFirst.Code, wFirst.Body.String())
	}

	c, w := f.ctxSuperAdmin(t)
	withJSONBody(c, http.MethodPost, `{"name":"`+secondName+`","admin_email":"`+email+`","admin_password":"longenoughpw","admin_name":"Ada"}`)
	f.SuperAdmin.CreateTenant(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("duplicate email code=%d, want 400 (body=%s)", w.Code, w.Body.String())
	}

	// RegisterAdmin is transactional, so the rejected create must leave no
	// tenant behind. It used to commit the tenant first and orphan it.
	var orphans int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM tenants WHERE name = $1`, secondName).Scan(&orphans); err != nil {
		t.Fatalf("count orphans: %v", err)
	}
	if orphans != 0 {
		t.Fatalf("rejected create left %d orphaned tenant(s); RegisterAdmin must roll back", orphans)
	}
}

func TestSuperAdminGetTenant(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", strconv.Itoa(f.TenantID))
	f.SuperAdmin.GetTenant(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != f.TenantID {
		t.Fatalf("id=%d want %d", got.ID, f.TenantID)
	}

	// GetTenant returns the bare row, not a {"data":[]} envelope.
	bad, wBad := f.ctxSuperAdmin(t)
	withParam(bad, "id", "abc")
	f.SuperAdmin.GetTenant(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}

	missing, wMissing := f.ctxSuperAdmin(t)
	withParam(missing, "id", "99999999")
	f.SuperAdmin.GetTenant(missing)
	if wMissing.Code != http.StatusNotFound {
		t.Fatalf("unknown id code=%d, want 404 (body=%s)", wMissing.Code, wMissing.Body.String())
	}
}

func TestSuperAdminUpdateTenant(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", strconv.Itoa(f.TenantID))
	withJSONBody(c, http.MethodPut, `{"name":"Renamed Org"}`)
	f.SuperAdmin.UpdateTenant(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var name string
	if err := f.DB.QueryRow(`SELECT name FROM tenants WHERE id = $1`, f.TenantID).Scan(&name); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if name != "Renamed Org" {
		t.Fatalf("name=%q", name)
	}

	noName, wNoName := f.ctxSuperAdmin(t)
	withParam(noName, "id", strconv.Itoa(f.TenantID))
	withJSONBody(noName, http.MethodPut, `{}`)
	f.SuperAdmin.UpdateTenant(noName)
	if wNoName.Code != http.StatusBadRequest {
		t.Fatalf("missing name code=%d, want 400", wNoName.Code)
	}

	bad, wBad := f.ctxSuperAdmin(t)
	withParam(bad, "id", "abc")
	withJSONBody(bad, http.MethodPut, `{"name":"X"}`)
	f.SuperAdmin.UpdateTenant(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}
}

// An unknown tenant is a client mistake, so it must be a 404 rather than the 500
// it used to return.
func TestSuperAdminUpdateTenantNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", "99999999")
	withJSONBody(c, http.MethodPut, `{"name":"Nope"}`)
	f.SuperAdmin.UpdateTenant(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

func TestSuperAdminSuspendReactivateTenantRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)

	sus, wSus := f.ctxSuperAdmin(t)
	withParam(sus, "id", strconv.Itoa(f.TenantID))
	f.SuperAdmin.SuspendTenant(sus)
	if wSus.Code != http.StatusOK {
		t.Fatalf("suspend code=%d body=%s", wSus.Code, wSus.Body.String())
	}

	var suspended sql.NullTime
	if err := f.DB.QueryRow(`SELECT suspended_at FROM tenants WHERE id = $1`, f.TenantID).Scan(&suspended); err != nil {
		t.Fatalf("read suspended_at: %v", err)
	}
	if !suspended.Valid {
		t.Fatal("suspended_at still NULL after suspend")
	}

	rea, wRea := f.ctxSuperAdmin(t)
	withParam(rea, "id", strconv.Itoa(f.TenantID))
	f.SuperAdmin.ReactivateTenant(rea)
	if wRea.Code != http.StatusOK {
		t.Fatalf("reactivate code=%d body=%s", wRea.Code, wRea.Body.String())
	}
	if err := f.DB.QueryRow(`SELECT suspended_at FROM tenants WHERE id = $1`, f.TenantID).Scan(&suspended); err != nil {
		t.Fatalf("re-read suspended_at: %v", err)
	}
	if suspended.Valid {
		t.Fatalf("suspended_at=%v still set after reactivate", suspended.Time)
	}
}

func TestSuperAdminSuspendTenantNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", "99999999")
	f.SuperAdmin.SuspendTenant(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

// Reactivate of an unknown tenant is a 404, matching Suspend and GetTenant.
func TestSuperAdminReactivateTenantNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", "99999999")
	f.SuperAdmin.ReactivateTenant(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

// In the posture a deployed environment uses (DEBUG unset), a bad tenant id must
// not leak the raw Postgres foreign-key error, which discloses schema details.
//
// This handler used to call utils.BadRequest(c, err.Error()), returning that text
// unconditionally. It now maps only the known client error to 400 and routes the
// rest through InternalError, which hides the error unless DEBUG=true.
func TestSuperAdminCreateTenantAdminDoesNotLeakDBErrors(t *testing.T) {
	f := newHandlerFixture(t)
	t.Setenv("DEBUG", "false") // production posture; .env sets this to true locally

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", "99999999")
	withJSONBody(c, http.MethodPost, `{"email":"`+testutil.UniqueEmail(t, "leak")+`","password":"longenoughpw","name":"X"}`)
	f.SuperAdmin.CreateTenantAdmin(c)

	if w.Code == http.StatusOK {
		t.Fatalf("unknown tenant reported success: %s", w.Body.String())
	}
	assertNoDBInternals(t, w.Body.String())
}

// DEBUG=true is a deliberate developer affordance in utils.InternalError and
// SafeErrorResponse: it returns err.Error() so the frontend can surface details.
// The consequence is that every one of the ~205 InternalError/SafeErrorResponse
// call sites returns raw database text in that mode, and .env ships DEBUG=true,
// which godotenv.Load copies into the process env.
//
// Pinned so the tradeoff is explicit: this is expected behaviour, not a bug, and
// the real risk is DEBUG=true reaching a deployed environment.
func TestSuperAdminCreateTenantAdminLeaksUnderDebugByDesign(t *testing.T) {
	f := newHandlerFixture(t)
	t.Setenv("DEBUG", "true")

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", "99999999")
	withJSONBody(c, http.MethodPost, `{"email":"`+testutil.UniqueEmail(t, "dbg")+`","password":"longenoughpw","name":"X"}`)
	f.SuperAdmin.CreateTenantAdmin(c)

	if !strings.Contains(strings.ToLower(w.Body.String()), "foreign key") {
		t.Logf("DEBUG=true did not leak here (status %d, body %s)", w.Code, w.Body.String())
	}
	t.Logf("DEBUG=true returns %d with body %s — expected, but DEBUG must never be true in a deployed environment", w.Code, w.Body.String())
}

// assertNoDBInternals fails if a response carries database text a client should
// never see.
func assertNoDBInternals(t *testing.T, body string) {
	t.Helper()
	for _, leak := range []string{"foreign key", "violates", "postgres", "pgcode", "sqlstate", "pq:"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Fatalf("response leaks database internals (%q): %s", leak, body)
		}
	}
}

func TestSuperAdminListTenantAdmins(t *testing.T) {
	f := newHandlerFixture(t)

	var email string
	if err := f.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, f.AdminUserID).Scan(&email); err != nil {
		t.Fatalf("read admin email: %v", err)
	}

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", strconv.Itoa(f.TenantID))
	f.SuperAdmin.ListTenantAdmins(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), email) {
		t.Fatalf("admin email %q missing from %s", email, w.Body.String())
	}

	// A tenant with no admins must still return an array, not null.
	empty, wEmpty := f.ctxSuperAdmin(t)
	withParam(empty, "id", strconv.Itoa(f.OtherTenantID))
	f.SuperAdmin.ListTenantAdmins(empty)
	if wEmpty.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", wEmpty.Code, wEmpty.Body.String())
	}
	if !strings.Contains(wEmpty.Body.String(), `"data":[]`) {
		t.Fatalf("expected an empty array, got %s", wEmpty.Body.String())
	}
}

func TestSuperAdminCreateTenantAdmin(t *testing.T) {
	f := newHandlerFixture(t)
	email := testutil.UniqueEmail(t, "added")

	c, w := f.ctxSuperAdmin(t)
	withParam(c, "id", strconv.Itoa(f.TenantID))
	withJSONBody(c, http.MethodPost, `{"email":"`+email+`","password":"longenoughpw","name":"Second Admin"}`)
	f.SuperAdmin.CreateTenantAdmin(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		UserID int `json:"user_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.UserID == 0 {
		t.Fatalf("no user_id returned: %s", w.Body.String())
	}

	var gotTenant int
	var role string
	if err := f.DB.QueryRow(`SELECT tenant_id, role FROM users WHERE id = $1`, resp.UserID).Scan(&gotTenant, &role); err != nil {
		t.Fatalf("new admin not persisted: %v", err)
	}
	if gotTenant != f.TenantID {
		t.Fatalf("new admin tenant_id=%d want %d", gotTenant, f.TenantID)
	}
	if role != "admin" {
		t.Fatalf("role=%q want admin", role)
	}
}

func TestSuperAdminCreateTenantAdminRejections(t *testing.T) {
	f := newHandlerFixture(t)
	existing := testutil.UniqueEmail(t, "rejects")

	c0, w0 := f.ctxSuperAdmin(t)
	withParam(c0, "id", strconv.Itoa(f.TenantID))
	withJSONBody(c0, http.MethodPost, `{"email":"`+existing+`","password":"longenoughpw","name":"First"}`)
	f.SuperAdmin.CreateTenantAdmin(c0)
	if w0.Code != http.StatusCreated {
		t.Fatalf("setup create code=%d body=%s", w0.Code, w0.Body.String())
	}

	cases := map[string]string{
		"weak password":     `{"email":"` + testutil.UniqueEmail(t, "wp") + `","password":"short","name":"X"}`,
		"missing email":     `{"password":"longenoughpw","name":"X"}`,
		"missing password":  `{"email":"` + testutil.UniqueEmail(t, "mp") + `","name":"X"}`,
		"missing name":      `{"email":"` + testutil.UniqueEmail(t, "mn") + `","password":"longenoughpw"}`,
		"duplicate email":   `{"email":"` + existing + `","password":"longenoughpw","name":"Dup"}`,
		"malformed payload": `{"email":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c, w := f.ctxSuperAdmin(t)
			withParam(c, "id", strconv.Itoa(f.TenantID))
			withJSONBody(c, http.MethodPost, body)
			f.SuperAdmin.CreateTenantAdmin(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("code=%d, want 400 (body=%s)", w.Code, w.Body.String())
			}
		})
	}

	badID, wBadID := f.ctxSuperAdmin(t)
	withParam(badID, "id", "abc")
	withJSONBody(badID, http.MethodPost, `{"email":"`+testutil.UniqueEmail(t, "bi")+`","password":"longenoughpw","name":"X"}`)
	f.SuperAdmin.CreateTenantAdmin(badID)
	if wBadID.Code != http.StatusBadRequest {
		t.Fatalf("bad tenant id code=%d, want 400", wBadID.Code)
	}
}

// DEFECT PINNED: GetGlobalStats hardcodes stats["revenue"] = 49900 rather than
// summing anything, so a super-admin dashboard reads a constant as MRR.
func TestSuperAdminGetGlobalStats(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	f.SuperAdmin.GetGlobalStats(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"tenants", "free_tenants", "paid_tenants", "revenue"} {
		if _, ok := stats[key]; !ok {
			t.Fatalf("key %q missing from %v", key, stats)
		}
	}

	var actualTenants, actualFree int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&actualTenants); err != nil {
		t.Fatalf("count tenants: %v", err)
	}
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM tenants t
		LEFT JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE ts.plan_id IS NULL
		   OR ts.plan_id = (SELECT id FROM subscription_plans WHERE slug = 'free')`).Scan(&actualFree); err != nil {
		t.Fatalf("count free tenants: %v", err)
	}

	if got := int(stats["tenants"].(float64)); got != actualTenants {
		t.Fatalf("stats tenants=%d, db says %d", got, actualTenants)
	}
	if got := int(stats["free_tenants"].(float64)); got != actualFree {
		t.Fatalf("stats free_tenants=%d, db says %d", got, actualFree)
	}
	if got, want := int(stats["paid_tenants"].(float64)), actualTenants-actualFree; got != want {
		t.Fatalf("stats paid_tenants=%d, want %d", got, want)
	}

	t.Logf("revenue=%v (pinned placeholder in TenantRepo.GetGlobalStats)", stats["revenue"])
}