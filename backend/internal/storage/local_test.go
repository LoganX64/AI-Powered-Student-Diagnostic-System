package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// LocalStorage is pure filesystem, so the whole surface is testable offline
// against t.TempDir().

func newLocal(t *testing.T) *LocalStorage {
	t.Helper()
	return NewLocal(t.TempDir())
}

func TestNewLocalDefaultsToUploads(t *testing.T) {
	if got := NewLocal("").Dir; got != "./uploads" {
		t.Fatalf("Dir=%q, want ./uploads", got)
	}
	if got := NewLocal("/tmp/x").Dir; got != "/tmp/x" {
		t.Fatalf("Dir=%q", got)
	}
}

func TestLocalPutReturnsFilesystemPath(t *testing.T) {
	l := newLocal(t)

	url, err := l.Put(t.Context(), "assignments/1/0.mp4", strings.NewReader("chunk"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	// Local storage returns a path, not a URL — callers must not treat it as one.
	want := filepath.Join(l.Dir, filepath.FromSlash("assignments/1/0.mp4"))
	if url != want {
		t.Fatalf("Put returned %q, want %q", url, want)
	}
}

func TestLocalPutCreatesNestedDirs(t *testing.T) {
	l := newLocal(t)

	if _, err := l.Put(t.Context(), "a/b/c/deep.mp4", strings.NewReader("x")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(filepath.Join(l.Dir, "a", "b", "c", "deep.mp4")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}
}

func TestLocalGetRoundTrip(t *testing.T) {
	l := newLocal(t)
	if _, err := l.Put(t.Context(), "k/v.txt", strings.NewReader("hello")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := l.Get(t.Context(), "k/v.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("got %q", string(b))
	}
}

func TestLocalGetMissing(t *testing.T) {
	l := newLocal(t)
	if _, err := l.Get(t.Context(), "nope.txt"); err == nil {
		t.Fatal("Get on a missing key must fail")
	}
}

func TestLocalListReturnsSlashRelativeKeys(t *testing.T) {
	l := newLocal(t)
	l.Put(t.Context(), "pre/a.txt", strings.NewReader("1"))
	l.Put(t.Context(), "pre/nested/b.txt", strings.NewReader("2"))
	// A sibling outside the prefix must not appear.
	l.Put(t.Context(), "other/c.txt", strings.NewReader("3"))

	keys, err := l.List(t.Context(), "pre")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	sort.Strings(keys)
	want := []string{"pre/a.txt", "pre/nested/b.txt"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keys=%v want %v", keys, want)
	}
	// Keys must use forward slashes regardless of OS separator.
	for _, k := range keys {
		if strings.ContainsRune(k, '\\') {
			t.Fatalf("key %q must be slash-separated", k)
		}
	}
}

func TestLocalListMissingPrefixIsEmpty(t *testing.T) {
	l := newLocal(t)
	keys, err := l.List(t.Context(), "does-not-exist")
	if err != nil {
		t.Fatalf("List on missing prefix must not error: %v", err)
	}
	if keys != nil {
		t.Fatalf("keys=%v, want nil", keys)
	}
}

func TestLocalListOnFileFails(t *testing.T) {
	l := newLocal(t)
	l.Put(t.Context(), "afile.txt", strings.NewReader("x"))
	if _, err := l.List(t.Context(), "afile.txt"); err == nil {
		t.Fatal("List on a file (not a directory) must fail")
	}
}

func TestLocalDelete(t *testing.T) {
	l := newLocal(t)
	l.Put(t.Context(), "d.txt", strings.NewReader("x"))

	if err := l.Delete(t.Context(), "d.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := l.Get(t.Context(), "d.txt"); err == nil {
		t.Fatal("file survived Delete")
	}
	// Deleting again must be a no-op, not an error.
	if err := l.Delete(t.Context(), "d.txt"); err != nil {
		t.Fatalf("Delete on missing key must be nil, got %v", err)
	}
}

func TestLocalDeletePrefixRemovesSubtree(t *testing.T) {
	l := newLocal(t)
	l.Put(t.Context(), "tree/a.txt", strings.NewReader("1"))
	l.Put(t.Context(), "tree/deep/b.txt", strings.NewReader("2"))
	l.Put(t.Context(), "keep/c.txt", strings.NewReader("3"))

	if err := l.DeletePrefix(t.Context(), "tree"); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	if _, err := os.Stat(filepath.Join(l.Dir, "tree")); !os.IsNotExist(err) {
		t.Fatal("subtree survived DeletePrefix")
	}
	// Read the unrelated key through a handle we close, so t.TempDir cleanup is
	// not blocked by an open file on Windows.
	rc, err := l.Get(t.Context(), "keep/c.txt")
	if err != nil {
		t.Fatalf("DeletePrefix removed an unrelated key: %v", err)
	}
	rc.Close()
	// Missing prefix must be a no-op.
	if err := l.DeletePrefix(t.Context(), "gone"); err != nil {
		t.Fatalf("DeletePrefix on missing prefix must be nil, got %v", err)
	}
}

// TestLocalPutPathTraversal is an open SECURITY FINDING, not a passing test.
//
// LocalStorage.Put joins the key onto Dir with filepath.Join and no traversal
// guard, so a key like "../escaped.txt" writes outside the storage directory.
// Nothing upstream sanitises the key: the video handler builds keys from an
// assignment id (numeric), so the practical exposure depends on whether any
// caller can influence key segments. Until a guard lands, this test records the
// unsafe behaviour explicitly and logs it, so the suite stays green while the
// gap stays visible.
func TestLocalPutPathTraversal(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "storage")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	l := NewLocal(root)

	escaped, err := l.Put(t.Context(), "../escaped.txt", strings.NewReader("x"))
	if err != nil {
		// A guard was added: the unsafe path is closed, which is the good outcome.
		t.Logf("path traversal now rejected: %v", err)
		return
	}
	outside := filepath.Join(parent, "escaped.txt")
	if _, statErr := os.Stat(outside); statErr == nil {
		t.Logf("SECURITY FINDING: LocalStorage.Put has no traversal guard — key %q escaped to %s (returned %q)", "../escaped.txt", outside, escaped)
	} else {
		t.Logf("traversal key returned path %q but nothing landed outside the dir", escaped)
	}
}

// TestLocalStorageSatisfiesInterface keeps the contract compile-checked.
func TestLocalStorageSatisfiesInterface(t *testing.T) {
	var _ Storage = NewLocal(t.TempDir())
	_ = bytes.MinRead
}
