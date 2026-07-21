package store

import (
	"errors"
	"sync"

	"go-crud-api/models"
)

type SessionStore struct {
	mu sync.RWMutex
	sessions map[string]models.Session
}

func NewSessionStore() *SessionStore {
	return  &SessionStore{
		sessions: make(map[string]models.Session),
	}
}

func (s *SessionStore) Create(session models.Session) models.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return  session
}

func (s *SessionStore) GetByID(id string) (models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[id]
	if !exists {
		return models.Session{}, errors.New("session not found")
	}
	return session , nil
}


func (s *SessionStore) UpdateStatus(id string, status models.SessionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, exists := s.sessions[id]
	if !exists {
		return errors.New("Session not found")
	}
	session.Status = status
	s.sessions[id] = session
	return nil
}

func (s *SessionStore) IncementFrameCount(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[id]
	session.FrameCount++
	s.sessions[id] = session
}

func (s *SessionStore) GetAll() []models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]models.Session, 0, len(s.sessions))

	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return  sessions
}
