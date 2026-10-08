package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// coach_ops.go had no direct tests. UpdateCoach is the only handler method that
// runs an explicit transaction — Begin, then UpdateName + UpdateEmail +
// DeleteCoachSubjects + CreateCoachSubjectsInTx, then Commit, with a deferred
// Rollback — so these tests concentrate on the all-or-nothing behaviour and the
// branches that leave the transaction untouched.

func coachExistsActive(t *testing.T, f *handlerFixture, coachID, tenantID int) bool {
	t.Helper()
	var deleted sql.NullTime
	if err := f.DB.QueryRow(`SELECT deleted_at FROM coaches WHERE id = $1 AND tenant_id = $2`,
		coachID, tenantID).Scan(&deleted); err != nil {
		t.Fatalf("read coach %d: %v", coachID, err)
	}
	return !deleted.Valid
}

func TestUpdateCoachRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)
	second := testutil.CreateSubject(t, f.DB, f.TenantID, "Second Subject")
	testutil.LinkCoachSubject(t, f.DB, f.CoachID, f.SubjectID)

	newEmail := testutil.UniqueEmail(t, "renamed")
	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	withJSONBody(c, http.MethodPut, `{"name":"Renamed Coach","email":"`+newEmail+`","subject_ids":[`+
		strconv.Itoa(second)+`]}`)

	f.Admin.UpdateCoach(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	// All three writes land: coaches.name, users.email, and the subject links.
	var name string
	if err := f.DB.QueryRow(`SELECT name FROM coaches WHERE id = $1`, f.CoachID).Scan(&name); err != nil {
		t.Fatalf("read coach name: %v", err)
	}
	if name != "Renamed Coach" {
		t.Fatalf("coach name=%q want %q", name, "Renamed Coach")
	}

	var gotEmail string
	if err := f.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, f.CoachUserID).Scan(&gotEmail); err != nil {
		t.Fatalf("read user email: %v", err)
	}
	if gotEmail != newEmail {
		t.Fatalf("user email=%q want %q", gotEmail, newEmail)
	}

	got := testutil.CoachSubjectIDs(t, f.DB, f.CoachID)
	if !reflect.DeepEqual(got, []int{second}) {
		t.Fatalf("subject links=%v want [%d]", got, second)
	}
}

// UpdateCoach deletes then recreates the subject links inside the transaction.
// A row-by-row append would leave the old link in place; this asserts the
// replacement actually happened.
func TestUpdateCoachReplacesSubjects(t *testing.T) {
	f := newHandlerFixture(t)
	kept := testutil.CreateSubject(t, f.DB, f.TenantID, "Kept")
	replacement := testutil.CreateSubject(t, f.DB, f.TenantID, "Replacement")
	testutil.LinkCoachSubject(t, f.DB, f.CoachID, f.SubjectID)
	testutil.LinkCoachSubject(t, f.DB, f.CoachID, kept)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	withJSONBody(c, http.MethodPut, `{"name":"Coach One","email":"`+testutil.UniqueEmail(t, "c")+`","subject_ids":[`+
		strconv.Itoa(replacement)+`]}`)

	f.Admin.UpdateCoach(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	got := testutil.CoachSubjectIDs(t, f.DB, f.CoachID)
	want := []int{replacement}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subject links=%v want %v — the old links were not deleted", got, want)
	}
	for _, gone := range []int{f.SubjectID, kept} {
		for _, id := range got {
			if id == gone {
				t.Fatalf("stale subject link %d survived: %v", gone, got)
			}
		}
	}
}

// An empty subject_ids list takes the len(req.SubjectIDs) > 0 false branch: the
// existing links are deleted and nothing is recreated.
func TestUpdateCoachEmptySubjectIDsClearsLinks(t *testing.T) {
	f := newHandlerFixture(t)
	testutil.LinkCoachSubject(t, f.DB, f.CoachID, f.SubjectID)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	withJSONBody(c, http.MethodPut, `{"name":"Coach One","email":"`+testutil.UniqueEmail(t, "c")+`","subject_ids":[]}`)

	f.Admin.UpdateCoach(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if got := testutil.CoachSubjectIDs(t, f.DB, f.CoachID); len(got) != 0 {
		t.Fatalf("subject links=%v, want empty", got)
	}
}

// The duplicate-email check returns before Begin, so nothing is written — not
// just the email, but the name and the subject links too.
func TestUpdateCoachDuplicateEmail(t *testing.T) {
	f := newHandlerFixture(t)
	other := testutil.CreateSubject(t, f.DB, f.TenantID, "Other Subject")
	testutil.LinkCoachSubject(t, f.DB, f.CoachID, f.SubjectID)

	// Another user's email, which EmailExistsForOther must report as taken.
	taken := testutil.UniqueEmail(t, "taken")
	if _, err := f.DB.Exec(`UPDATE users SET email = $1 WHERE id = $2`, taken, f.AdminUserID); err != nil {
		t.Fatalf("claim email: %v", err)
	}

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	withJSONBody(c, http.MethodPut, `{"name":"Should Not Apply","email":"`+taken+`","subject_ids":[`+
		strconv.Itoa(other)+`]}`)

	f.Admin.UpdateCoach(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400 (body=%s)", w.Code, w.Body.String())
	}

	var name string
	if err := f.DB.QueryRow(`SELECT name FROM coaches WHERE id = $1`, f.CoachID).Scan(&name); err != nil {
		t.Fatalf("read coach name: %v", err)
	}
	if name != "Coach One" {
		t.Fatalf("coach name changed to %q despite the rejected update", name)
	}
	got := testutil.CoachSubjectIDs(t, f.DB, f.CoachID)
	if !reflect.DeepEqual(got, []int{f.SubjectID}) {
		t.Fatalf("subject links=%v, want the original [%d]", got, f.SubjectID)
	}
}

func TestUpdateCoachBadPayload(t *testing.T) {
	f := newHandlerFixture(t)
	cases := map[string]string{
		"missing name":        `{"email":"a@b.co","subject_ids":[1]}`,
		"missing email":       `{"name":"X","subject_ids":[1]}`,
		"missing subject_ids": `{"name":"X","email":"a@b.co"}`,
		"malformed json":      `{"name":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c, w := f.ctx(t, "admin")
			withParam(c, "id", strconv.Itoa(f.CoachID))
			withJSONBody(c, http.MethodPut, body)
			f.Admin.UpdateCoach(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("code=%d, want 400 (body=%s)", w.Code, w.Body.String())
			}
		})
	}

	bad, wBad := f.ctx(t, "admin")
	withParam(bad, "id", "abc")
	withJSONBody(bad, http.MethodPut, `{"name":"X","email":"a@b.co","subject_ids":[1]}`)
	f.Admin.UpdateCoach(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}
}

// A coach in another tenant must be invisible, so the update 404s on
// GetDetail before the transaction starts.
func TestUpdateCoachNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.OtherCoachID))
	withJSONBody(c, http.MethodPut, `{"name":"Hijacked","email":"`+testutil.UniqueEmail(t, "hijack")+`","subject_ids":[1]}`)
	f.Admin.UpdateCoach(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}

	var name string
	if err := f.DB.QueryRow(`SELECT name FROM coaches WHERE id = $1`, f.OtherCoachID).Scan(&name); err != nil {
		t.Fatalf("read foreign coach: %v", err)
	}
	if name != "Coach One" {
		t.Fatalf("foreign coach renamed to %q", name)
	}
}

func TestDeleteReactivateCoachRoundTrip(t *testing.T) {
	f := newHandlerFixture(t)

	del, wDel := f.ctx(t, "admin")
	withParam(del, "id", strconv.Itoa(f.CoachID))
	f.Admin.DeleteCoach(del)
	if wDel.Code != http.StatusOK {
		t.Fatalf("delete code=%d body=%s", wDel.Code, wDel.Body.String())
	}
	if coachExistsActive(t, f, f.CoachID, f.TenantID) {
		t.Fatal("coach still active after delete")
	}

	// Hidden from the default listing...
	hiddenBody, _ := adminListCoaches(t, f, false)
	if bodyContainsCoachID(hiddenBody, f.CoachID) {
		t.Fatalf("deleted coach still in the default listing: %s", hiddenBody)
	}
	// ...but present among the deactivated.
	deactivatedBody, _ := adminListCoaches(t, f, true)
	if !bodyContainsCoachID(deactivatedBody, f.CoachID) {
		t.Fatalf("deleted coach missing from include_deactivated listing: %s", deactivatedBody)
	}

	rea, wRea := f.ctx(t, "admin")
	withParam(rea, "id", strconv.Itoa(f.CoachID))
	f.Admin.ReactivateCoach(rea)
	if wRea.Code != http.StatusOK {
		t.Fatalf("reactivate code=%d body=%s", wRea.Code, wRea.Body.String())
	}
	if !coachExistsActive(t, f, f.CoachID, f.TenantID) {
		t.Fatal("coach still deactivated after reactivate")
	}
	reactivatedBody, _ := adminListCoaches(t, f, false)
	if !bodyContainsCoachID(reactivatedBody, f.CoachID) {
		t.Fatalf("reactivated coach missing from the default listing: %s", reactivatedBody)
	}
}

// A second delete finds nothing to deactivate and reports "already deactivated".
func TestDeleteCoachTwice(t *testing.T) {
	f := newHandlerFixture(t)

	first, wFirst := f.ctx(t, "admin")
	withParam(first, "id", strconv.Itoa(f.CoachID))
	f.Admin.DeleteCoach(first)
	if wFirst.Code != http.StatusOK {
		t.Fatalf("first delete code=%d", wFirst.Code)
	}

	second, wSecond := f.ctx(t, "admin")
	withParam(second, "id", strconv.Itoa(f.CoachID))
	f.Admin.DeleteCoach(second)
	if wSecond.Code != http.StatusNotFound {
		t.Fatalf("second delete code=%d, want 404 (body=%s)", wSecond.Code, wSecond.Body.String())
	}
}

// Reactivate on an already-active coach is the mirror case.
func TestReactivateActiveCoach(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	f.Admin.ReactivateCoach(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

// A cross-tenant coach must survive both operations untouched.
func TestDeleteCoachCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.OtherCoachID))
	f.Admin.DeleteCoach(c)

	if w.Code == http.StatusOK {
		t.Fatalf("deleted another tenant's coach: %s", w.Body.String())
	}
	if !coachExistsActive(t, f, f.OtherCoachID, f.OtherTenantID) {
		t.Fatal("foreign coach was deactivated")
	}
}

func TestListCoachTestsAndStudents(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	f.Admin.ListCoachTests(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ListCoachTests code=%d body=%s", w.Code, w.Body.String())
	}
	if !containsStr(w.Body.String(), "Algebra Paper") {
		t.Fatalf("own coach's test missing: %s", w.Body.String())
	}
	if containsStr(w.Body.String(), "Foreign Paper") {
		t.Fatalf("another tenant's test leaked: %s", w.Body.String())
	}

	s, wS := f.ctx(t, "admin")
	withParam(s, "id", strconv.Itoa(f.CoachID))
	f.Admin.ListCoachStudents(s)
	if wS.Code != http.StatusOK {
		t.Fatalf("ListCoachStudents code=%d body=%s", wS.Code, wS.Body.String())
	}
	if !containsStr(wS.Body.String(), "Primary Student") {
		t.Fatalf("own coach's student missing: %s", wS.Body.String())
	}
	if containsStr(wS.Body.String(), "Foreign Student") {
		t.Fatalf("another tenant's student leaked: %s", wS.Body.String())
	}

	// A foreign coach id fails verifyCoachExists, so the route 404s.
	bad, wBad := f.ctx(t, "admin")
	withParam(bad, "id", strconv.Itoa(f.OtherCoachID))
	f.Admin.ListCoachStudents(bad)
	if wBad.Code != http.StatusNotFound {
		t.Fatalf("foreign coach id code=%d, want 404 (body=%s)", wBad.Code, wBad.Body.String())
	}

	badTests, wBadTests := f.ctx(t, "admin")
	withParam(badTests, "id", strconv.Itoa(f.OtherCoachID))
	f.Admin.ListCoachTests(badTests)
	if wBadTests.Code != http.StatusNotFound {
		t.Fatalf("foreign coach id code=%d, want 404 (body=%s)", wBadTests.Code, wBadTests.Body.String())
	}
}

func TestGetCoachStatsBatch(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withJSONBody(c, http.MethodPost, `{"coach_ids":[`+strconv.Itoa(f.CoachID)+`]}`)
	f.Admin.GetCoachStatsBatch(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			CoachID     int     `json:"coach_id"`
			StudentCount int    `json:"student_count"`
			AverageSQI  float64 `json:"average_sqi"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data=%+v, want one metric for our coach", resp.Data)
	}
	if resp.Data[0].CoachID != f.CoachID {
		t.Fatalf("coach_id=%d want %d", resp.Data[0].CoachID, f.CoachID)
	}
	if resp.Data[0].StudentCount != 1 {
		t.Fatalf("student_count=%d want 1", resp.Data[0].StudentCount)
	}
}

// The super_admin refusal runs before ShouldBindJSON, so it holds even for a
// malformed body.
func TestGetCoachStatsBatchRefusesSuperAdmin(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctxSuperAdmin(t)
	withJSONBody(c, http.MethodPost, `{"coach_ids":[`+strconv.Itoa(f.CoachID)+`]}`)
	f.Admin.GetCoachStatsBatch(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestGetCoachStatsBatchValidation(t *testing.T) {
	f := newHandlerFixture(t)

	tooMany := make([]string, maxBatchCoachIDs+1)
	for i := range tooMany {
		tooMany[i] = strconv.Itoa(i + 1)
	}

	cases := map[string]struct {
		body string
		want int
	}{
		"missing coach_ids": {body: `{}`, want: http.StatusBadRequest},
		"empty array":       {body: `{"coach_ids":[]}`, want: http.StatusBadRequest},
		"over the cap":      {body: `{"coach_ids":[` + joinInts(tooMany) + `]}`, want: http.StatusBadRequest},
		"malformed json":    {body: `{"coach_ids":`, want: http.StatusBadRequest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, w := f.ctx(t, "admin")
			withJSONBody(c, http.MethodPost, tc.body)
			f.Admin.GetCoachStatsBatch(c)
			if w.Code != tc.want {
				t.Fatalf("code=%d, want %d (body=%s)", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

// The cap must accept exactly maxBatchCoachIDs, not reject it.
func TestGetCoachStatsBatchAtCap(t *testing.T) {
	f := newHandlerFixture(t)

	ids := make([]string, maxBatchCoachIDs)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}
	c, w := f.ctx(t, "admin")
	withJSONBody(c, http.MethodPost, `{"coach_ids":[`+joinInts(ids)+`]}`)
	f.Admin.GetCoachStatsBatch(c)

	if w.Code != http.StatusOK {
		t.Fatalf("exactly %d ids rejected: code=%d body=%s", maxBatchCoachIDs, w.Code, w.Body.String())
	}
}

func TestGetCoach(t *testing.T) {
	f := newHandlerFixture(t)

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.CoachID))
	f.Admin.GetCoach(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	bad, wBad := f.ctx(t, "admin")
	withParam(bad, "id", "abc")
	f.Admin.GetCoach(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}

	// Any repo error maps to 404 in this handler, including a real cross-tenant
	// lookup that returns nil, nil.
	foreign, wForeign := f.ctx(t, "admin")
	withParam(foreign, "id", strconv.Itoa(f.OtherCoachID))
	f.Admin.GetCoach(foreign)
	if wForeign.Code != http.StatusNotFound {
		t.Fatalf("foreign coach code=%d, want 404 (body=%s)", wForeign.Code, wForeign.Body.String())
	}
}

func TestListCoachesScopedToTenant(t *testing.T) {
	f := newHandlerFixture(t)
	var foreignEmail string
	if err := f.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, f.OtherCoachUserID).Scan(&foreignEmail); err != nil {
		t.Fatalf("read foreign coach email: %v", err)
	}

	body, code := adminListCoaches(t, f, false)
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if containsStr(body, foreignEmail) {
		t.Fatalf("listing leaked another tenant's coach: %s", body)
	}

	// search narrows within our own tenant.
	body, code = adminListCoachesSearch(t, f, "Coach")
	if code != http.StatusOK {
		t.Fatalf("search code=%d body=%s", code, body)
	}
	if !containsStr(body, "Coach One") {
		t.Fatalf("search=Coach did not match our coach: %s", body)
	}
}

func adminListCoaches(t *testing.T, f *handlerFixture, includeDeactivated bool) (string, int) {
	t.Helper()
	return adminListCoachesQuery(t, f, "include_deactivated="+boolQuery(includeDeactivated))
}

func adminListCoachesSearch(t *testing.T, f *handlerFixture, search string) (string, int) {
	t.Helper()
	return adminListCoachesQuery(t, f, "search="+urlEscape(search))
}

func adminListCoachesQuery(t *testing.T, f *handlerFixture, query string) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/coaches?"+query, nil)
	f.Admin.ListCoaches(c)
	return w.Body.String(), w.Code
}

func boolQuery(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func joinInts(parts []string) string { return strings.Join(parts, ",") }

func containsStr(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// bodyContainsCoachID checks a coach listing payload. CoachRepo.List emits a
// `coach_id` column, not `id`, so an assertion keyed on the wrong field would
// silently return false for every row — which is exactly the bug this caught.
func bodyContainsCoachID(body string, id int) bool {
	var resp struct {
		Data []struct {
			CoachID int `json:"coach_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return false
	}
	for _, d := range resp.Data {
		if d.CoachID == id {
			return true
		}
	}
	return false
}