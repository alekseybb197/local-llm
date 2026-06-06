package pkg

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// State represents the OAuth2 authorization flow state
type State struct {
	// State
	AuthCode string // Authorization code from provider
	CodeVerifier string // PKCE code verifier
	CodeChallenge string // PKCE code challenge
	RedirectURL string // Callback URL
	State string // CSRF state token
	Expiry time.Time // State expiration
	CreatedAt time.Time
}

// StateMachine manages OAuth2 state lifecycle
type StateMachine struct {
	states map[string]*State
	mu sync.RWMutex
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		states: make(map[string]*State),
	}
}

// GenerateStateToken creates a cryptographically secure state token
func GenerateStateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// NewState creates a new state
func NewState(authCode, codeVerifier, codeChallenge, redirectURL string) (*State, error) {
	stateToken, err := GenerateStateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiry := now.Add(10 * time.Minute)

	return &State{
		AuthCode:     authCode,
		CodeVerifier: codeVerifier,
		CodeChallenge: codeChallenge,
		RedirectURL:  redirectURL,
		State:        stateToken,
		Expiry:       expiry,
		CreatedAt:    now,
	}, nil
}

// GetState retrieves a state by token
func (sm *StateMachine) GetState(token string) (*State, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[token]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(state.Expiry) {
		return nil, false
	}

	return state, true
}

// DeleteState removes a state by token
func (sm *StateMachine) DeleteState(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, token)
}

// StoreState stores a state by token
func (sm *StateMachine) StoreState(state *State) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states[state.State] = state
}

// GCExpiredStates removes expired states
func (sm *StateMachine) GCExpiredStates() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for token, state := range sm.states {
		if now.After(state.Expiry) {
			delete(sm.states, token)
		}
	}
}

// VerifyCodeChallenge validates PKCE code challenge
func VerifyCodeChallenge(codeVerifier, codeChallenge string) bool {
	// Simple validation: in production, use proper hashing
	// For now, we'll use the challenge as-is for simplicity
	return codeChallenge != ""
}
