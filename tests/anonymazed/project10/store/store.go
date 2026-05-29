package store

import (
	"sync"
	"time"

	"oauth2proxy/models"
)

// SessionStore provides session management
type SessionStore interface {
	StoreSession(session *models.Session) error
	GetSession(stateToken string) (*models.Session, error)
	ClearSession(stateToken string) error
}

// sessionStore implementation
type sessionStore struct {
	store map[string]*models.Session
	mu    sync.RWMutex
}

func NewSessionStore() SessionStore {
	return &sessionStore{
		store: make(map[string]*models.Session),
	}
}

func (s *sessionStore) StoreSession(session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[session.StateToken] = session
	return nil
}

func (s *sessionStore) GetSession(stateToken string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.store[stateToken]
	if !ok {
		return nil, &NotFoundError{}
	}
	return session, nil
}

func (s *sessionStore) ClearSession(stateToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, stateToken)
	return nil
}

// UserStore provides user management
type UserStore interface {
	GetUserByCode(code string, stateToken string) (*models.User, error)
}

// userStore implementation
type userStore struct {
	store map[string]*models.User
	mu    sync.RWMutex
}

func NewUserStore() UserStore {
	return &userStore{
		store: make(map[string]*models.User),
	}
}

func (s *userStore) GetUserByCode(code string, stateToken string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if code == "" {
		return nil, &NotFoundError{}
	}

	user, ok := s.store[code]
	if !ok {
		// Create new user for development
		user = &models.User{
			ID:           code,
			AccessToken:  "dev-access-token",
			RefreshToken: "dev-refresh-token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		s.store[code] = user
	}

	return user, nil
}

// NotFoundError is returned when a resource is not found
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}
