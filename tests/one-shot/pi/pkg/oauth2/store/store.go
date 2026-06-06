package store

import (
	"time"
)

// Session is an OAuth2 session
type Session struct {
	UserID    string
	CodeVerifier string
	CreatedAt time.Time
}

// SessionStore stores OAuth2 sessions
type SessionStore interface {
	// Get retrieves a session
	Get(sessionID string) (*Session, error)
	// Store stores a session
	Store(sessionID string, session *Session) error
	// Delete deletes a session
	Delete(sessionID string) error
	// CleanExpired cleans expired sessions
	CleanExpired() error
}

// inMemorySessionStore implements SessionStore
type inMemorySessionStore struct {
	store map[string]*Session
}

func NewInMemorySessionStore() *inMemorySessionStore {
	return &inMemorySessionStore{
		store: make(map[string]*Session),
	}
}

func (s *inMemorySessionStore) Get(sessionID string) (*Session, error) {
	session, exists := s.store[sessionID]
	if !exists {
		return nil, nil
	}
	return session, nil
}

func (s *inMemorySessionStore) Store(sessionID string, session *Session) error {
	s.store[sessionID] = session
	return nil
}

func (s *inMemorySessionStore) Delete(sessionID string) error {
	delete(s.store, sessionID)
	return nil
}

func (s *inMemorySessionStore) CleanExpired() error {
	// Clean expired sessions
	now := time.Now()
	for id, session := range s.store {
		if session.CreatedAt.Add(15 * time.Minute).Before(now) {
			delete(s.store, id)
		}
	}
	return nil
}

// Store is the interface for OAuth2 stores
type Store interface {
	// User operations
	GetOrCreateUser(username, name, email string) (*User, error)

	// OAuth2 Token operations
	ValidateToken(token, scope string) (*OAuth2Token, error)
	CreateToken(userID, accessToken, refreshToken, tokenType, scope string, expiresIn int) (*OAuth2Token, error)
	DeleteToken(token string) error

	// API Key operations
	ListAPIKeys() (*APIKeyList, error)
	GetAPIKey(id string) (*APIKey, error)
	CreateAPIKey(name, role, scope string) (*APIKey, error)
	DeleteAPIKey(id string) error
	VerifyAPIKey(key string) (string, error)
}
