package models

import "time"

// SessionStatus represents the lifecycle of an exam session.
// Using a custom string type means you cannot accidentally pass
// an arbitrary string where a status is expected — the compiler catches it.

type SessionStatus string

const (
	StatusWaiting SessionStatus = "waiting"
	StatusActive SessionStatus = "active"
	StatusDisconnected SessionStatus = "disconnected"
	StatusComplete SessionStatus = "complete"
)

// Session represents one student's exam session.
// In the real proctoring system this maps to a database row.
// For now it lives in memory.

type Session struct {
	ID string `json:"id"`
	StudentID string `json:"student_id"`
	ExamID string `json:"exam_id"`
	Status SessionStatus `json:"status"`
	StartedAt time.Time `json:"started_at"`
	FrameCount int `json:"frame_count"`
	FlagCount int `json:"flag_count"`
}

// Frame represents a single video frame received from the Rust agent.
// In production this carries raw JPEG bytes.
// For learning we carry a text message instead.

type Frame struct {
	SessionID string `json:"session_id"`
	FrameNum int `json:"frame_num"`
	Timestamp int64 `json:"timestamp"`
	Data string `json:"data"`

}
// FrameResult is what comes back after the ML service analyzes a frame.
type FrameResult struct {
	SessionID string `json:"session_id"`
	FrameNum int `json:"frame_num"`
	Suspicious bool `json:"suspicious"`
	Confidence float64 `json:"confidence"`
}