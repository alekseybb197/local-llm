package main

import (
	"sync"
	"time"
)

type TokenStore struct {
	mu   sync.RWMutex
	tokens map[string]*TokenRecord
}

type TokenRecord struct {
	Subject string
	Scopes  []string
	Expiry  time.Time
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*TokenRecord),
	}
}

func (s *TokenStore) Get(token string) (*TokenRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.tokens[token]
	return record, ok
}

func (s *TokenStore) Set(token string, record *TokenRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = record
}

func (s *TokenStore) CleanUp() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, record := range s.tokens {
		if record.Expiry.Before(now) {
			delete(s.tokens, token)
		}
	}
}
