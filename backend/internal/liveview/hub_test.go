package liveview

import (
	"context"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// The hub's bookkeeping is exercised offline. RegisterStudent/AddViewer write to
// and close real websocket conns and spawn ping goroutines, so they are
// deliberately not driven here; everything below needs no network and no Redis.

func TestIsLiveAbsentSession(t *testing.T) {
	h := NewHub(nil)
	if h.IsLive(42) {
		t.Fatal("no session must not be live")
	}
}

func TestIsLiveRequiresAFrame(t *testing.T) {
	h := NewHub(nil)
	// A registered session that has never relayed a frame is not live.
	h.sessions[7] = &Session{Viewers: map[*websocket.Conn]struct{}{}}
	if h.IsLive(7) {
		t.Fatal("session with no frame yet must not be live")
	}
}

func TestIsLiveAfterRelayFrame(t *testing.T) {
	h := NewHub(nil)
	h.sessions[7] = &Session{Viewers: map[*websocket.Conn]struct{}{}}

	h.RelayFrame(7, []byte("frame-bytes"))
	if !h.IsLive(7) {
		t.Fatal("session must be live immediately after a frame")
	}
}

func TestIsLiveGoesStale(t *testing.T) {
	h := NewHub(nil)
	sess := &Session{Viewers: map[*websocket.Conn]struct{}{}, LastFrameAt: time.Now()}
	h.sessions[7] = sess

	// IsLive's window is 5 seconds.
	sess.LastFrameAt = time.Now().Add(-6 * time.Second)
	if h.IsLive(7) {
		t.Fatal("session must not be live after the 5s window")
	}
}

func TestRelayFrameStoresLatestFrame(t *testing.T) {
	h := NewHub(nil)
	sess := &Session{Viewers: map[*websocket.Conn]struct{}{}}
	h.sessions[1] = sess

	h.RelayFrame(1, []byte("first"))
	h.RelayFrame(1, []byte("second"))

	sess.FrameMu.RLock()
	defer sess.FrameMu.RUnlock()
	if string(sess.LatestFrame) != "second" {
		t.Fatalf("LatestFrame=%q, want the most recent frame", string(sess.LatestFrame))
	}
}

func TestRelayFrameUnknownSessionIsNoOp(t *testing.T) {
	h := NewHub(nil)
	// Must not panic when the student has already disconnected.
	h.RelayFrame(999, []byte("orphan"))
}

func TestRelayFrameWithNilRedisDoesNotPanic(t *testing.T) {
	h := NewHub(nil)
	h.sessions[1] = &Session{Viewers: map[*websocket.Conn]struct{}{}}
	// The publish is guarded by `h.rdb != nil`; this asserts that guard holds.
	h.RelayFrame(1, []byte("x"))
}

func TestRemoveViewerUnknownSessionIsNoOp(t *testing.T) {
	h := NewHub(nil)
	// RemoveViewer dereferences nothing on the conn (it is only a map key), so a
	// zero-value conn is safe here.
	h.RemoveViewer(12345, &websocket.Conn{})
}

func TestRemoveViewerDeletesFromSet(t *testing.T) {
	h := NewHub(nil)
	v1, v2 := &websocket.Conn{}, &websocket.Conn{}
	sess := &Session{Viewers: map[*websocket.Conn]struct{}{v1: {}, v2: {}}}
	h.sessions[5] = sess

	h.RemoveViewer(5, v1)

	sess.ViewerMu.RLock()
	defer sess.ViewerMu.RUnlock()
	if _, ok := sess.Viewers[v1]; ok {
		t.Fatal("viewer must be removed from the set")
	}
	if _, ok := sess.Viewers[v2]; !ok {
		t.Fatal("unrelated viewer must remain")
	}
}

func TestUnregisterStudentRemovesSession(t *testing.T) {
	h := NewHub(nil)
	// Viewers is left empty: UnregisterStudent calls viewer.Close(), which
	// dereferences the underlying net.Conn, so a session with viewers would
	// need a real websocket handshake. The behaviour under test is the session
	// teardown itself.
	sess := &Session{
		Viewers: map[*websocket.Conn]struct{}{},
		Cancel:  func() {},
	}
	h.sessions[11] = sess

	h.UnregisterStudent(11)
	if h.IsLive(11) {
		t.Fatal("session must be gone after UnregisterStudent")
	}
	if _, ok := h.sessions[11]; ok {
		t.Fatal("session must be deleted from the map")
	}
}

func TestUnregisterStudentUnknownIsNoOp(t *testing.T) {
	h := NewHub(nil)
	h.UnregisterStudent(4242) // must not panic
}

func TestAddViewerWithoutSessionReturnsErrNoSession(t *testing.T) {
	h := NewHub(nil)
	if err := h.AddViewer(1, &websocket.Conn{}); err != ErrNoSession {
		t.Fatalf("err=%v, want ErrNoSession", err)
	}
}

func TestSubscribeRedisNilClientReturnsImmediately(t *testing.T) {
	h := NewHub(nil)
	ctx, cancel := contextWithTimeout(50 * time.Millisecond)
	defer cancel()
	// With rdb nil the method returns before touching the network.
	h.subscribeRedis(ctx, 1)
}
