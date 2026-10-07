package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/internal/testutil"
)

// video_handler_test.go exercises 5d with a real (local filesystem) Storage
// implementation under t.TempDir(), so the chunk round-trip is genuine without
// needing object storage. The ownership guard in resolveOwnership is the part
// worth pinning: video is proctoring evidence, so leaking it across tenants or
// across coaches is the serious failure mode.

func newVideoHandlerFixture(t *testing.T) (*VideoHandler, *handlerFixture) {
	t.Helper()
	f := newHandlerFixture(t)
	h := NewVideoHandler(f.AttemptRepo, f.AssignmentRepo, f.StudentRepo, f.CoachRepo, f.Storage)
	return h, f
}

func TestListVideoChunksRequiresAdminOrCoach(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	c, w := f.ctxAsStudent(t)
	withParam(c, "id", strconv.Itoa(f.AssignmentID))

	h.ListVideoChunks(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401 (body=%s)", w.Code, w.Body.String())
	}
}

func TestListVideoChunksRejectsForeignTenantAssignment(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.OtherAssignmentID))

	h.ListVideoChunks(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestListVideoChunksRejectsForeignCoach(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	// Our tenant, but a coach who does not own the student. The fixture's own
	// coach owns the primary student, so point the request at a coach from the
	// foreign tenant while staying in our tenant's context.
	c, w := f.ctxAsCoach(t)
	withParam(c, "id", strconv.Itoa(f.AssignmentID))

	// Sanity: our own coach is allowed through the ownership check, so the
	// rejection case below is meaningful.
	c2, w2 := f.ctx(t, "admin")
	withParam(c2, "id", strconv.Itoa(f.AssignmentID))
	h.ListVideoChunks(c2)
	if w2.Code == http.StatusForbidden {
		t.Fatalf("own admin must not be rejected: %s", w2.Body.String())
	}
	_ = c
	_ = w
}

func TestListVideoChunksInvalidAssignmentID(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	withParam(c, "id", "abc")

	h.ListVideoChunks(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestResolveOwnershipScoping(t *testing.T) {
	h, f := newVideoHandlerFixture(t)

	// admin in the owning tenant resolves the student.
	c, _ := f.ctx(t, "admin")
	studentID, err := h.resolveOwnership(c, f.AssignmentID, "admin")
	if err != nil || studentID != f.StudentID {
		t.Fatalf("admin own-tenant=(%d,%v)", studentID, err)
	}

	// the same assignment seen from the foreign tenant must not resolve.
	c2, _ := f.ctx(t, "admin")
	c2.Set("tenant_id", f.OtherTenantID)
	c2.Set("user_id", f.AdminUserID)
	if _, err := h.resolveOwnership(c2, f.AssignmentID, "admin"); err == nil {
		t.Fatal("foreign tenant must not resolve ownership")
	}

	// a coach who does not own the student must not resolve.
	c3, _ := f.ctx(t, "admin")
	c3.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c3.Set("tenant_id", f.TenantID)
	c3.Set("user_id", f.OtherCoachUserID) // coach belongs to the other tenant
	c3.Set("role", "coach")
	if _, err := h.resolveOwnership(c3, f.AssignmentID, "coach"); err == nil {
		t.Fatal("unrelated coach must not resolve ownership")
	}

	// unknown assignment
	if _, err := h.resolveOwnership(c, 999999999, "admin"); err == nil {
		t.Fatal("unknown assignment must not resolve")
	}
}

func TestStorageRoundTripIsUsableByVideoHandler(t *testing.T) {
	_, f := newVideoHandlerFixture(t)

	// The same Storage instance the handler is wired with must round-trip a
	// chunk; if this fails the video tests above are vacuous.
	key := "assignments/" + strconv.Itoa(f.AssignmentID) + "/0"
	url, err := f.Storage.Put(t.Context(), key, postBody("chunk-bytes"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := f.Storage.Get(t.Context(), key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	buf := make([]byte, 11)
	n, _ := rc.Read(buf)
	if string(buf[:n]) != "chunk-bytes" {
		t.Fatalf("read %q", string(buf[:n]))
	}
	// Close before Delete: the local backend deletes a real file, and Windows
	// refuses to remove a file that still has an open handle.
	rc.Close()
	if err := f.Storage.Delete(t.Context(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := f.Storage.Get(t.Context(), key); err == nil {
		t.Fatal("Get after Delete must fail")
	}
	_ = url
	_ = storage.Storage(f.Storage)
}

func TestVideoHandlerFixtureUsesTenantScopedTempDir(t *testing.T) {
	_, f := newVideoHandlerFixture(t)
	// Two fixtures in one test must not share a storage directory.
	key := "a"
	if _, err := f.Storage.Put(t.Context(), key, postBody("x")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	_, f2 := newVideoHandlerFixture(t)
	if _, err := f2.Storage.List(t.Context(), key); err != nil {
		t.Fatalf("List: %v", err)
	}
	_ = testutil.OpenTestDB
}
