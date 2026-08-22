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

const (
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

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
	FrameMu     sync.RWMutex
	StartedAt   time.Time
	LastFrameAt time.Time
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:      rdb,
		sessions: make(map[int]*Session),
	}
}

func (h *Hub) RegisterStudent(studentID int, conn *websocket.Conn) {
	h.mu.Lock()

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
	h.mu.Unlock()

	go h.subscribeRedis(ctx, studentID)
	go h.pingStudent(conn)

	log.Printf("[LIVEVIEW] Student %d registered, session started", studentID)
}

func (h *Hub) UnregisterStudent(studentID int) {
	h.mu.Lock()
	sess, ok := h.sessions[studentID]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.sessions, studentID)
	h.mu.Unlock()

	sess.Cancel()

	sess.ViewerMu.RLock()
	viewers := make([]*websocket.Conn, 0, len(sess.Viewers))
	for viewer := range sess.Viewers {
		viewers = append(viewers, viewer)
	}
	sess.ViewerMu.RUnlock()

	for _, viewer := range viewers {
		viewer.Close()
	}

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

	sess.FrameMu.RLock()
	latest := sess.LatestFrame
	sess.FrameMu.RUnlock()

	if len(latest) > 0 {
		if err := viewerConn.WriteMessage(websocket.BinaryMessage, latest); err != nil {
			log.Printf("[LIVEVIEW] Failed to send initial frame to viewer: %v", err)
			h.RemoveViewer(studentID, viewerConn)
			viewerConn.Close()
			return err
		}
	}

	go h.pingViewer(viewerConn)

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
	count := len(sess.Viewers)
	sess.ViewerMu.Unlock()

	log.Printf("[LIVEVIEW] Viewer removed for student %d (%d viewers)", studentID, count)
}

func (h *Hub) IsLive(studentID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sess, ok := h.sessions[studentID]
	if !ok {
		return false
	}
	sess.FrameMu.RLock()
	defer sess.FrameMu.RUnlock()
	return !sess.LastFrameAt.IsZero() && time.Since(sess.LastFrameAt) < 5*time.Second
}

func (h *Hub) RelayFrame(studentID int, frame []byte) {
	h.mu.RLock()
	sess, ok := h.sessions[studentID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	sess.FrameMu.Lock()
	sess.LatestFrame = frame
	sess.LastFrameAt = time.Now()
	sess.FrameMu.Unlock()

	sess.ViewerMu.RLock()
	badViewers := make([]*websocket.Conn, 0)
	for viewer := range sess.Viewers {
		if err := viewer.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			log.Printf("[LIVEVIEW] Failed to send frame to viewer: %v", err)
			badViewers = append(badViewers, viewer)
		}
	}
	sess.ViewerMu.RUnlock()

	if len(badViewers) > 0 {
		sess.ViewerMu.Lock()
		for _, v := range badViewers {
			delete(sess.Viewers, v)
			v.Close()
		}
		sess.ViewerMu.Unlock()
	}

	if h.rdb != nil {
		h.rdb.Publish(context.Background(), "liveview:student:"+strconv.Itoa(studentID), frame)
	}
}

func (h *Hub) subscribeRedis(ctx context.Context, studentID int) {
	if h.rdb == nil {
		return
	}
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

			h.mu.RLock()
			sess, ok := h.sessions[studentID]
			h.mu.RUnlock()
			if !ok {
				return
			}

			frame := []byte(msg.Payload)

			sess.FrameMu.Lock()
			sess.LatestFrame = frame
			sess.FrameMu.Unlock()

			sess.ViewerMu.RLock()
			for viewer := range sess.Viewers {
				if err := viewer.WriteMessage(websocket.BinaryMessage, frame); err != nil {
					viewer.Close()
				}
			}
			sess.ViewerMu.RUnlock()
		}
	}
}

func (h *Hub) pingStudent(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}

func (h *Hub) pingViewer(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}
