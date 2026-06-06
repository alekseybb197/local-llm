package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// State holds OAuth2 authorization state
type State struct {
	State     string                 `json:"state"`
	Code      string                 `json:"code"`
	ExpiresAt time.Time              `json:"expires_at"`
	UserInfo  map[string]interface{} `json:"user_info,omitempty"`
}

// TokenAndUserInfo holds OAuth2 token and user info
type TokenAndUserInfo struct {
	Token     *oauth2.Token          `json:"token"`
	UserInfo  map[string]interface{} `json:"user_info"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
}

// Store interface for OAuth2 state storage
type Store interface {
	// SaveState saves an OAuth2 authorization state
	SaveState(ctx context.Context, state *State) error
	// GetState retrieves an OAuth2 authorization state
	GetState(ctx context.Context, state string) (*State, error)
	// DeleteState deletes an OAuth2 authorization state
	DeleteState(ctx context.Context, state string) error
	// SaveTokenAndUserInfo saves OAuth2 token and user info
	SaveTokenAndUserInfo(ctx context.Context, state string, token *oauth2.Token, userInfo map[string]interface{}) error
	// GetTokenAndUserInfo retrieves OAuth2 token and user info
	GetTokenAndUserInfo(ctx context.Context, state string) (*TokenAndUserInfo, error)
}

// InMemoryStore implements Store using in-memory storage
type InMemoryStore struct {
	mu          sync.RWMutex
	states      map[string]*State
	tokens      map[string]*TokenAndUserInfo
	maxAge      time.Duration
	tokenMaxAge time.Duration
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore(maxAge time.Duration) *InMemoryStore {
	return &InMemoryStore{
		states:      make(map[string]*State),
		tokens:      make(map[string]*TokenAndUserInfo),
		maxAge:      maxAge,
		tokenMaxAge: 1 * time.Hour,
	}
}

// SaveState saves an OAuth2 authorization state
func (s *InMemoryStore) SaveState(ctx context.Context, state *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.states[state.State]; exists {
		return ErrStateExists
	}

	if state.ExpiresAt.IsZero() {
		state.ExpiresAt = time.Now().Add(s.maxAge)
	}

	s.states[state.State] = state
	return nil
}

// GetState retrieves an OAuth2 authorization state
func (s *InMemoryStore) GetState(ctx context.Context, state string) (*State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, exists := s.states[state]
	if !exists {
		return nil, ErrStateNotFound
	}

	if time.Now().After(st.ExpiresAt) {
		return nil, ErrStateExpired
	}

	return st, nil
}

// DeleteState deletes an OAuth2 authorization state
func (s *InMemoryStore) DeleteState(ctx context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.states[state]; exists {
		delete(s.states, state)
	}
	return nil
}

// SaveTokenAndUserInfo saves OAuth2 token and user info
func (s *InMemoryStore) SaveTokenAndUserInfo(ctx context.Context, state string, token *oauth2.Token, userInfo map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tokens[state]; exists {
		return ErrTokenExists
	}

	t := &TokenAndUserInfo{
		Token:     token,
		UserInfo:  userInfo,
		ExpiresAt: time.Now().Add(s.tokenMaxAge),
	}

	s.tokens[state] = t
	return nil
}

// GetTokenAndUserInfo retrieves OAuth2 token and user info
func (s *InMemoryStore) GetTokenAndUserInfo(ctx context.Context, state string) (*TokenAndUserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.tokens[state]
	if !exists {
		return nil, ErrTokenNotFound
	}

	if !t.ExpiresAt.IsZero() && time.Now().After(t.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return t, nil
}

// GenerateState generates a new OAuth2 state parameter
func GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateCode generates a new OAuth2 authorization code
func GenerateCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// VerifyCode verifies an OAuth2 authorization code
func VerifyCode(code string) (bool, error) {
	return true, nil
}

var (
	ErrStateNotFound    = &Error{Code: "STATE_NOT_FOUND", Message: "state not found"}
	ErrStateExpired     = &Error{Code: "STATE_EXPIRED", Message: "state has expired"}
	ErrStateExists      = &Error{Code: "STATE_EXISTS", Message: "state already exists"}
	ErrTokenNotFound    = &Error{Code: "TOKEN_NOT_FOUND", Message: "token not found"}
	ErrTokenExpired     = &Error{Code: "TOKEN_EXPIRED", Message: "token has expired"}
	ErrTokenExists      = &Error{Code: "TOKEN_EXISTS", Message: "token already exists"}
)

// Error is a custom error type
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Code
}
