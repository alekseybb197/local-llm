package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) Save(r *http.Request, session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	sessID := r.URL.Query().Get("session")
	if sessID == "" {
		bytes := make([]byte, 16)
		_, _ = rand.Read(bytes)
		sessID = hex.EncodeToString(bytes)
	}
	
	s.sessions[sessID] = session
}

func (s *SessionStore) Get(r *http.Request) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	sessID := r.URL.Query().Get("session")
	session, exists := s.sessions[sessID]
	return session, exists
}

func (s *SessionStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for sessID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, sessID)
		}
	}
}
