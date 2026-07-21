package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-crud-api/models"
	"go-crud-api/store"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

// SessionHandler manages exam sessions and WebSocket connections.
type SessionHandler struct {
	sessions *store.SessionStore
	// jobs is the shared channel all WebSocket handlers push frames into.
    // The worker pool on the other end pulls frames out and analyzes them.
    // Buffered at 256 — absorbs bursts without blocking the network handler.
	jobs chan frameJob
	wg sync.WaitGroup

	// connMu protects the conns map from concurrent access.
	// Multiple WebSocket handlers register and deregister simultaneously.
	connMu sync.RWMutex
	conns map[string]*websocket.Conn
}

// frameJob bundles a frame with its session ID so workers know
// which session to update after analysis.
type frameJob struct {
	frame models.Frame
	session models.Session
}

func NewSessionHandler(sessions *store.SessionStore) * SessionHandler {
	h := &SessionHandler{
		sessions: sessions,
		jobs: make(chan frameJob, 256),
		conns: make(map[string]*websocket.Conn),
	}
    // Start the worker pool when the handler is created.
    // 5 workers process frames concurrently.
    // In production this number is tuned based on ML service throughput.
	h.startWorkerPool(5)
	return h
}
// startWorkerPool launches N goroutines that process frames forever.
// They only stop when the jobs channel is closed — which happens on server shutdown.
func (h *SessionHandler) startWorkerPool(n int) {
	for i := range n {
		go func(workerID int) {
			for job := range h.jobs {
				h.analyzeFrame(workerID, job)
			}
		}(i + 1)
	}
}

// analyzeFrame simulates the gRPC call to the Python ML service.
// In Phase 3 of your roadmap this becomes a real gRPC client call.
func (h *SessionHandler) analyzeFrame(workerID int, job frameJob){
	// Simulate ML processing time
	time.Sleep((10 * time.Millisecond))

	// Mock result — flag every 7th frame as suspicious
	suspicious := job.frame.FrameNum%7 == 0

	h.sessions.IncementFrameCount(job.session.ID)
	if suspicious {
		h.sessions.IncementFrameCount(job.session.ID)
	}

	slog.Info("frame analyzed",
		"worker", workerID,
		"Session_id", job.session.ID,
		"frame-num", job.frame.FrameNum,
		"suspicious", suspicious,
	)
}

// CreateSession is a REST endpoint — POST /sessions
// The Rust agent calls this first to get a session ID,
// then opens a WebSocket connection using that ID.
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
        StudentID string `json:"student_id"`
        ExamID string `json:"exam_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.StudentID == "" || body.ExamID == "" {
		respondError(w, http.StatusBadRequest, "student_id and exam_id required")
		return
	}

	session := models.Session{
		ID: uuid.NewString(),
		StudentID: body.StudentID,
		ExamID: body.ExamID,
		Status: models.StatusWaiting,
		StartedAt: time.Now(),
	}

	created := h.sessions.Create(session)
	respond(w, http.StatusCreated, created)
}

// GetSession is a REST endpoint — GET /sessions/{id}
// The exam dashboard polls this to see frame counts and flags in real time.
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	session, err := h.sessions.GetByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, session)
}

// GetAllSessions - Get /sessions
func (h *SessionHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, h.sessions.GetAll())
}

// HandleWebSocket is where the real work happens — GET /sessions/{id}/stream
// The Rust agent upgrades the HTTP connection to WebSocket here
// and starts streaming frames.
func (h *SessionHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

    // Verify the session exists before accepting the WebSocket connection
	session, err := h.sessions.GetByID(sessionID) 
	if err != nil {
		respond(w, http.StatusNotFound, "session not found")
		return
	}
	// Accept upgrades the HTTP connection to a WebSocket connection.
    // InsecureSkipVerify disables origin checking — fine for development.
    // In production you validate the Origin header against your domain.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
    // CloseNow closes gracefully
	defer conn.CloseNow()
	// Register this session with the WaitGroup before doing any work.
	// Shutdown() will wait until every session calls wg.Done().
	h.wg.Add(1)
	defer h.wg.Done()

	// Register connection so shutdown can close it actively.
	h.registerConn(sessionID, conn)
	defer h.deregisterConn(sessionID)

	slog.Info("websocket connected", "session_id", sessionID)
	h.sessions.UpdateStatus(sessionID, models.StatusActive)

    // Run the session loop — this blocks until the client disconnects.
    // Everything happens inside here: receiving frames, pushing to workers,
    // sending acknowledgements back.
	h.runSessionLoop(r.Context(), conn, session)

    // When the loop exits, the client has disconnected.
	h.sessions.UpdateStatus(sessionID, models.StatusDisconnected)
    slog.Info("websocket disconnected", "session_id", sessionID)
}

// runSessionLoop is the heart of each exam session.
// It runs in the goroutine that HandleWebSocket was called in —
// which means each connected Rust agent has its own goroutine running this loop.
// 500 concurrent sessions = 500 goroutines running this function simultaneously.
func (h *SessionHandler) runSessionLoop(
	ctx context.Context,
	conn *websocket.Conn,
	session models.Session,
) {
	frameNum := 0

	for {
		var frame models.Frame
		err := wsjson.Read(ctx, conn, &frame)
		if err != nil {
			status := websocket.CloseStatus(err)
			switch status {
			case websocket.StatusNormalClosure:
				slog.Info("client disconnected normally", "session_id", session.ID)
			case websocket.StatusGoingAway:
				// This is us — server shutdown closed the connection intentionally.
				slog.Info("session closed for server shutdown", "session_id", session.ID)
			default:
				slog.Error("unexpected disconnect", "session_id", session.ID, "error", err)
			}
			return
		}

		frameNum++
		frame.SessionID = session.ID
		frame.FrameNum = frameNum
		frame.Timestamp = time.Now().UnixMilli()

		select {
		case h.jobs <- frameJob{frame: frame, session: session}:
			// frame queued successfully
		case <-ctx.Done():
			return
		}

		ack := map[string]any{
			"frame_num":   frameNum,
			"session_id":  session.ID,
			"received_at": time.Now().UnixMilli(),
		}
		if err := wsjson.Write(ctx, conn, ack); err != nil {
			slog.Error("failed to send ack", "session_id", session.ID, "error", err)
			return
		}
	}
}

// Shutdown drains the worker pool and waits for all active sessions to close.
// Called by main() after the OS shutdown signal arrives.
func(h *SessionHandler) Shutdown(ctx context.Context) {
	slog.Info("closing active sessions", "count", len(h.conns))

	// Close every active WebSocket connection with a proper close frame.
	// StatusGoingAway (1001) is the correct code for a server shutdown —
	// it tells the client "I am going away, please reconnect elsewhere."
	// The Rust agent will receive this, log it, and immediately attempt
	// to reconnect to the next available server instance.

	h.connMu.RLock()
	
	for sessionID, conn := range h.conns {
		conn.Close(websocket.StatusGoingAway, "server shutting down")
		slog.Info("closed session connection", "session_id", sessionID)
	}
	h.connMu.RUnlock()
	
	// Now wait for all HandleWebSocket goroutines to finish their cleanup —
	// updating session status, deregistering from the map, calling wg.Done().
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(h.jobs)
		close(done)
	}()

	select{
	case <- done:
		slog.Info("all sessions drained cleanly")
	case <- ctx.Done():
		slog.Warn("shutdown timeout reached — some sessions may not have drained")
	}
}

// registerConn adds a connection to the registry when a session starts.
func (h *SessionHandler) registerConn(sessionID string, conn *websocket.Conn) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.conns[sessionID] = conn
}

// deregisterConn removes a connection from the registry when a session ends.
// Called via defer in HandleWebSocket so it always runs on exit.
func (h *SessionHandler) deregisterConn(sessionID string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	delete(h.conns, sessionID)
}