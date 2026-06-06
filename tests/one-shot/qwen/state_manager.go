package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type StateManager struct {
	mu       sync.RWMutex
	states   map[string]*AuthState
	secret   string
	timeout  time.Duration
}

func NewStateManager(secret string, timeout time.Duration) *StateManager {
	return &StateManager{
		states:   make(map[string]*AuthState),
		secret:   secret,
		timeout:  timeout,
	}
}

func (sm *StateManager) GenerateState() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	bytes := make([]byte, 32)
	rand.Read(bytes)
	state := hex.EncodeToString(bytes)
	
	sm.states[state] = &AuthState{
		State:      state,
		Time:       time.Now(),
		RedirectURI: "",
	}
	
	return state
}

func (sm *StateManager) ValidateState(state, redirectURI string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	authState, exists := sm.states[state]
	if !exists {
		return "", ErrInvalidState
	}
	
	if time.Since(authState.Time) > sm.timeout {
		delete(sm.states, state)
		return "", ErrInvalidState
	}
	
	if authState.RedirectURI != redirectURI {
		return "", ErrInvalidState
	}
	
	authState.Time = time.Now()
	return authState.RedirectURI, nil
}

func (sm *StateManager) CleanupExpired() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for state, authState := range sm.states {
		if now.Sub(authState.Time) > sm.timeout {
			delete(sm.states, state)
		}
	}
}
