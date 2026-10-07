package storage

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cloudinary is lazy: NewCloudinary only parses, all HTTP happens per request.
// Every field is exported, so tests override BaseURL/Client and point at an
// httptest server.

func TestNewCloudinaryParsesURL(t *testing.T) {
	cs, err := NewCloudinary("cloudinary://key123:secret456@mycloud/videos")
	if err != nil {
		t.Fatalf("NewCloudinary: %v", err)
	}
	if cs.CloudName != "mycloud" || cs.APIKey != "key123" || cs.APISecret != "secret456" {
		t.Fatalf("cs=%+v", cs)
	}
	if cs.Folder != "videos" {
		t.Fatalf("Folder=%q", cs.Folder)
	}
	if cs.BaseURL != "https://api.cloudinary.com/v1_1/mycloud" {
		t.Fatalf("BaseURL=%q", cs.BaseURL)
	}
	if cs.Client == nil {
		t.Fatal("Client must be set")
	}
}

func TestNewCloudinaryRejectsBadURLs(t *testing.T) {
	cases := []string{
		"https://key:secret@cloud",       // wrong scheme
		"cloudinary://",                  // no host
		"cloudinary://cloud",             // no key/secret
		"://nope",                        // unparseable
	}
	for _, u := range cases {
		if _, err := NewCloudinary(u); err == nil {
			t.Errorf("NewCloudinary(%q) must fail", u)
		}
	}
}

// TestCloudinarySignGolden pins the signature algorithm: sha1 of sorted
// "k=v&..." with the secret appended, hex encoded.
func TestCloudinarySignGolden(t *testing.T) {
	cs := &CloudinaryStorage{APISecret: "s3cr3t"}

	// single param
	want := sha1.Sum([]byte("public_id=abc" + "s3cr3t"))
	if got := cs.sign(map[string]string{"public_id": "abc"}); got != fmt.Sprintf("%x", want) {
		t.Fatalf("sign=%s want %x", got, want)
	}

	// multiple params must be sorted, not map-ordered
	params := map[string]string{
		"timestamp": "1700000000",
		"folder":    "videos",
		"public_id": "chunk-0",
	}
	want2 := sha1.Sum([]byte("folder=videos&public_id=chunk-0&timestamp=1700000000" + "s3cr3t"))
	if got := cs.sign(params); got != fmt.Sprintf("%x", want2) {
		t.Fatalf("sign=%s want %x", got, want2)
	}
}

func TestCloudinaryBuildFileURL(t *testing.T) {
	cs := &CloudinaryStorage{CloudName: "mycloud"}
	if got := cs.buildFileURL("assignments/1/video.mp4"); got != "https://res.cloudinary.com/mycloud/video/upload/assignments/1/video" {
		t.Fatalf("url=%s", got)
	}

	cs.Folder = "videos"
	if got := cs.buildFileURL("clip.mp4"); got != "https://res.cloudinary.com/mycloud/video/upload/videos/clip" {
		t.Fatalf("url with folder=%s", got)
	}
}

func TestCloudinaryListAndDeletePrefixUnsupported(t *testing.T) {
	cs := &CloudinaryStorage{}
	if _, err := cs.List(t.Context(), "pre"); err == nil {
		t.Fatal("List must return an error")
	}
	if err := cs.DeletePrefix(t.Context(), "pre"); err == nil {
		t.Fatal("DeletePrefix must return an error")
	}
}

func TestCloudinaryPutUploadsAndPrefersSecureURL(t *testing.T) {
	var gotPath string
	var form map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		form = r.MultipartForm.Value
		w.Header().Set("Content-Type", "application/json")
		// Return both; the code must prefer secure_url.
		json.NewEncoder(w).Encode(map[string]string{
			"secure_url": "https://cdn.example/secure.mp4",
			"url":        "http://cdn.example/insecure.mp4",
		})
	}))
	defer srv.Close()

	cs := &CloudinaryStorage{
		CloudName: "mycloud", APIKey: "key", APISecret: "secret",
		Folder: "videos", BaseURL: srv.URL, Client: srv.Client(),
	}

	got, err := cs.Put(t.Context(), "assignments/1/clip.mp4", strings.NewReader("video-bytes"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got != "https://cdn.example/secure.mp4" {
		t.Fatalf("Put must prefer secure_url, got %q", got)
	}
	if gotPath != "/video/upload" {
		t.Fatalf("path=%q", gotPath)
	}
	for _, f := range []string{"api_key", "timestamp", "signature", "folder", "public_id"} {
		if len(form[f]) == 0 {
			t.Errorf("missing multipart field %q (form=%v)", f, form)
		}
	}
	if form["api_key"][0] != "key" {
		t.Fatalf("api_key=%v", form["api_key"])
	}
	if form["folder"][0] != "videos" {
		t.Fatalf("folder=%v", form["folder"])
	}
	// public_id must strip the extension and keep the path.
	if form["public_id"][0] != "assignments/1/clip" {
		t.Fatalf("public_id=%v", form["public_id"])
	}
}

func TestCloudinaryPutFallsBackToURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": "http://cdn.example/plain.mp4"})
	}))
	defer srv.Close()

	cs := &CloudinaryStorage{APIKey: "k", APISecret: "s", BaseURL: srv.URL, Client: srv.Client()}
	got, err := cs.Put(t.Context(), "c.mp4", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got != "http://cdn.example/plain.mp4" {
		t.Fatalf("got %q", got)
	}
}

func TestCloudinaryPutPropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad signature"))
	}))
	defer srv.Close()

	cs := &CloudinaryStorage{APIKey: "k", APISecret: "s", BaseURL: srv.URL, Client: srv.Client()}
	if _, err := cs.Put(t.Context(), "c.mp4", strings.NewReader("x")); err == nil {
		t.Fatal("Put must fail on non-200")
	}
}

func TestCloudinaryDeleteSendsSignedRequest(t *testing.T) {
	var gotPath string
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer srv.Close()

	cs := &CloudinaryStorage{APIKey: "k", APISecret: "s", Folder: "videos", BaseURL: srv.URL, Client: srv.Client()}
	if err := cs.Delete(t.Context(), "clip.mp4"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotPath != "/video/destroy" {
		t.Fatalf("path=%q", gotPath)
	}
	// Folder must be folded into public_id for destroy.
	if !strings.Contains(body, "public_id=videos%2Fclip") && !strings.Contains(body, "public_id=videos/clip") {
		t.Fatalf("body=%q must carry the folder-qualified public_id", body)
	}
}

func TestCloudinaryDeletePropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cs := &CloudinaryStorage{APIKey: "k", APISecret: "s", BaseURL: srv.URL, Client: srv.Client()}
	if err := cs.Delete(t.Context(), "missing.mp4"); err == nil {
		t.Fatal("Delete must fail on non-200")
	}
}

// TestCloudinaryGetIgnoresBaseURL records a design quirk: Get builds its URL
// with buildFileURL, which hardcodes https://res.cloudinary.com/... and does
// NOT use BaseURL. So Get cannot be redirected at a test server, and the only
// seam is Client's Transport. The stub below intercepts the request so the test
// never touches the network (a real Get to an unreachable host would hang for
// the client's full timeout).
func TestCloudinaryGetNon200(t *testing.T) {
	var gotURL string
	cs := &CloudinaryStorage{
		CloudName: "mycloud",
		BaseURL:   "http://127.0.0.1:1", // deliberately ignored by Get
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotURL = r.URL.String()
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader("denied")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		})},
	}

	if _, err := cs.Get(t.Context(), "c.mp4"); err == nil {
		t.Fatal("Get must fail on non-200")
	}
	if !strings.HasPrefix(gotURL, "https://res.cloudinary.com/mycloud/video/upload/") {
		t.Fatalf("Get used %q; note it ignores BaseURL and targets the real CDN host", gotURL)
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
