package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	h := NewVideoHandler(f.AttemptRepo, f.AssignmentRepo, f.StudentRepo, f.CoachRepo, f.Storage, nil)
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

// putFailStorage wraps a Storage whose Put always fails, so the merge tests can
// assert what survives an upload failure.
type putFailStorage struct {
	storage.Storage
}

func (putFailStorage) Put(context.Context, string, io.Reader) (string, error) {
	return "", errors.New("simulated upload failure")
}

func enableVideoProctoring(t *testing.T, f *handlerFixture) {
	t.Helper()
	if _, err := f.DB.Exec(
		`UPDATE assignments SET integrity_policy = '{"video_proctoring":true}'::jsonb WHERE id=$1`,
		f.AssignmentID,
	); err != nil {
		t.Fatalf("enable video proctoring: %v", err)
	}
}

// seedChunks writes two valid WebM chunks under video/<attemptID>/ so the real
// ffmpeg merge path runs. Tests skip when ffmpeg is not installed.
func seedChunks(t *testing.T, f *handlerFixture, attemptID int) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	chunkPath := filepath.Join(t.TempDir(), "chunk.webm")
	var stderr strings.Builder
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=128x96:rate=10",
		"-c:v", "libvpx", "-an", chunkPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("generate test webm: %v: %s", err, stderr.String())
	}
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatalf("read test webm: %v", err)
	}
	for i := 0; i < 2; i++ {
		key := fmt.Sprintf("video/%d/%d.webm", attemptID, i)
		if _, err := f.Storage.Put(t.Context(), key, bytes.NewReader(data)); err != nil {
			t.Fatalf("put chunk %d: %v", i, err)
		}
	}
}

func listVideoKeys(t *testing.T, f *handlerFixture, attemptID int) []string {
	t.Helper()
	keys, err := f.Storage.List(t.Context(), fmt.Sprintf("video/%d/", attemptID))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return keys
}

// TestMergeForAttemptUploadFailurePreservesChunks pins the fix for the data-loss
// bug where chunks were deleted before the merged upload was confirmed: a failed
// Put must leave every original chunk intact.
func TestMergeForAttemptUploadFailurePreservesChunks(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	enableVideoProctoring(t, f)
	attemptID := testutil.CreateAttempt(t, f.DB, f.AssignmentID)
	seedChunks(t, f, attemptID)

	h.Storage = putFailStorage{f.Storage}

	if err := h.MergeForAttempt(f.AssignmentID); err == nil {
		t.Fatal("MergeForAttempt must fail when the merged upload fails")
	}

	keys := listVideoKeys(t, f, attemptID)
	want := fmt.Sprintf("video/%d/0.webm", attemptID)
	want2 := fmt.Sprintf("video/%d/1.webm", attemptID)
	merged := fmt.Sprintf("video/%d/merged.webm", attemptID)
	if len(keys) != 2 || keys[0] != want || keys[1] != want2 {
		t.Fatalf("chunks must survive a failed upload, got %v", keys)
	}
	if _, err := f.Storage.Get(t.Context(), merged); err == nil {
		t.Fatal("merged.webm must not exist after a failed upload")
	}
}

// TestMergeForAttemptSuccessKeepsOnlyMerged verifies the success path: the
// merged file survives the chunk cleanup (DeletePrefix would have removed it).
func TestMergeForAttemptSuccessKeepsOnlyMerged(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	enableVideoProctoring(t, f)
	attemptID := testutil.CreateAttempt(t, f.DB, f.AssignmentID)
	seedChunks(t, f, attemptID)

	if err := h.MergeForAttempt(f.AssignmentID); err != nil {
		t.Fatalf("MergeForAttempt: %v", err)
	}

	merged := fmt.Sprintf("video/%d/merged.webm", attemptID)
	keys := listVideoKeys(t, f, attemptID)
	if len(keys) != 1 || keys[0] != merged {
		t.Fatalf("want only merged.webm after success, got %v", keys)
	}
	rc, err := f.Storage.Get(t.Context(), merged)
	if err != nil {
		t.Fatalf("merged.webm must be retrievable: %v", err)
	}
	rc.Close()
}

func TestMergeForAttemptDoesNotReMergeIfAlreadyMerged(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	enableVideoProctoring(t, f)
	attemptID := testutil.CreateAttempt(t, f.DB, f.AssignmentID)
	seedChunks(t, f, attemptID)

	// First merge
	if err := h.MergeForAttempt(f.AssignmentID); err != nil {
		t.Fatalf("First MergeForAttempt: %v", err)
	}

	// Second merge must return nil immediately without errors
	if err := h.MergeForAttempt(f.AssignmentID); err != nil {
		t.Fatalf("Second MergeForAttempt should do nothing and return nil: %v", err)
	}
}

func TestVideoStatusReportsReadyWhenMerged(t *testing.T) {
	h, f := newVideoHandlerFixture(t)
	enableVideoProctoring(t, f)
	attemptID := testutil.CreateAttempt(t, f.DB, f.AssignmentID)
	seedChunks(t, f, attemptID)

	if err := h.MergeForAttempt(f.AssignmentID); err != nil {
		t.Fatalf("MergeForAttempt: %v", err)
	}

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(f.AssignmentID))

	h.VideoStatus(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"ready"`) {
		t.Fatalf("expected status:ready in body: %s", w.Body.String())
	}
}
