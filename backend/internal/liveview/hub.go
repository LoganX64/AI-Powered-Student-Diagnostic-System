package liveview

import (
	"context"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var ErrNoSession = errors.New("no active session for student")

type Hub struct {
	rdb      *redis.Client
	sessions map[int]*Session
	mu       sync.RWMutex
}

type Session struct {
	StudentConn *websocket.Conn
	Viewers     map[*websocket.Conn]struct{}
	ViewerMu    sync.RWMutex
	Cancel      context.CancelFunc
	LatestFrame []byte
	StartedAt   time.Time
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:      rdb,
		sessions: make(map[int]*Session),
	}
}

func (h *Hub) RegisterStudent(studentID int, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.sessions[studentID]; ok {
		existing.Cancel()
		existing.StudentConn.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess := &Session{
		StudentConn: conn,
		Viewers:     make(map[*websocket.Conn]struct{}),
		Cancel:      cancel,
		StartedAt:   time.Now(),
	}
	h.sessions[studentID] = sess

	go h.subscribeRedis(ctx, studentID)
	log.Printf("[LIVEVIEW] Student %d registered, session started", studentID)
}

func (h *Hub) UnregisterStudent(studentID int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sess, ok := h.sessions[studentID]
	if !ok {
		return
	}

	sess.Cancel()
	sess.StudentConn.Close()

	sess.ViewerMu.RLock()
	for viewer := range sess.Viewers {
		viewer.Close()
	}
	sess.ViewerMu.RUnlock()

	delete(h.sessions, studentID)
	log.Printf("[LIVEVIEW] Student %d unregistered, session cleaned up", studentID)
}

func (h *Hub) AddViewer(studentID int, viewerConn *websocket.Conn) error {
	h.mu.RLock()
	sess, ok := h.sessions[studentID]
	h.mu.RUnlock()

	if !ok {
		return ErrNoSession
	}

	sess.ViewerMu.Lock()
	sess.Viewers[viewerConn] = struct{}{}
	sess.ViewerMu.Unlock()

	if len(sess.LatestFrame) > 0 {
		viewerConn.WriteMessage(websocket.BinaryMessage, sess.LatestFrame)
	}

	log.Printf("[LIVEVIEW] Viewer added for student %d (%d viewers)", studentID, len(sess.Viewers))
	return nil
}

func (h *Hub) RemoveViewer(studentID int, viewerConn *websocket.Conn) {
	h.mu.RLock()
	sess, ok := h.sessions[studentID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	sess.ViewerMu.Lock()
	delete(sess.Viewers, viewerConn)
	sess.ViewerMu.Unlock()

	log.Printf("[LIVEVIEW] Viewer removed for student %d (%d viewers)", studentID, len(sess.Viewers))
}

func (h *Hub) IsLive(studentID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.sessions[studentID]
	return ok
}

func (h *Hub) RelayFrame(studentID int, frame []byte) {
	h.mu.RLock()
	sess, ok := h.sessions[studentID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	sess.LatestFrame = frame

	sess.ViewerMu.RLock()
	for viewer := range sess.Viewers {
		if err := viewer.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			log.Printf("[LIVEVIEW] Failed to send frame to viewer: %v", err)
			viewer.Close()
		}
	}
	sess.ViewerMu.RUnlock()

	h.rdb.Publish(context.Background(), "liveview:student:"+strconv.Itoa(studentID), frame)
}

func (h *Hub) subscribeRedis(ctx context.Context, studentID int) {
	pubsub := h.rdb.Subscribe(ctx, "liveview:student:"+strconv.Itoa(studentID))
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			pubsub.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_ = msg
		}
	}
}
