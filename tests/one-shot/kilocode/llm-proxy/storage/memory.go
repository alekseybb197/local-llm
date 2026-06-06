package storage

import (
	"sync"
	"time"

	"llm-proxy/models"
)

type Store interface {
	SaveAuthCode(code *models.OAuth2AuthCode) error
	GetAuthCode(code string) (*models.OAuth2AuthCode, error)
	RemoveAuthCode(code string) error
	SaveToken(token *models.OAuth2Token) error
	GetToken(token string, clientID string) (*models.OAuth2Token, error)
	RemoveToken(token string) error
	CleanExpired()
}

type MemoryStore struct {
	mu sync.RWMutex
	// Map: code -> *models.OAuth2AuthCode
	authCodes map[string]*models.OAuth2AuthCode
	// Map: token -> *models.OAuth2Token
	tokens map[string]*models.OAuth2Token
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		authCodes: make(map[string]*models.OAuth2AuthCode),
		tokens:    make(map[string]*models.OAuth2Token),
	}
}

func (s *MemoryStore) SaveAuthCode(code *models.OAuth2AuthCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authCodes[code.Code] = code
	return nil
}

func (s *MemoryStore) GetAuthCode(code string) (*models.OAuth2AuthCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	codeObj, exists := s.authCodes[code]
	if !exists {
		return nil, &AuthCodeNotFoundError{}
	}
	if codeObj.Used {
		return nil, &AuthCodeUsedError{}
	}
	if time.Now().After(codeObj.ExpiresAt) {
		return nil, &AuthCodeExpiredError{}
	}
	return codeObj, nil
}

func (s *MemoryStore) RemoveAuthCode(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.authCodes[code]; exists {
		delete(s.authCodes, code)
		return nil
	}
	return &AuthCodeNotFoundError{}
}

func (s *MemoryStore) SaveToken(token *models.OAuth2Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.AccessToken] = token
	return nil
}

func (s *MemoryStore) GetToken(token string, clientID string) (*models.OAuth2Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokenObj, exists := s.tokens[token]
	if !exists {
		return nil, &TokenNotFoundError{}
	}
	if tokenObj.ClientID != clientID {
		return nil, &TokenInvalidClientError{}
	}
	if time.Now().After(tokenObj.ExpiresAt) {
		return nil, &TokenExpiredError{}
	}
	return tokenObj, nil
}

func (s *MemoryStore) RemoveToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tokens[token]; exists {
		delete(s.tokens, token)
		return nil
	}
	return &TokenNotFoundError{}
}

func (s *MemoryStore) CleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Clean expired auth codes
	for code, authCode := range s.authCodes {
		if now.After(authCode.ExpiresAt) {
			delete(s.authCodes, code)
		}
	}

	// Clean expired tokens
	for token, tokenObj := range s.tokens {
		if now.After(tokenObj.ExpiresAt) {
			delete(s.tokens, token)
		}
	}
}

// Errors
type AuthCodeNotFoundError struct{}
type AuthCodeUsedError struct{}
type AuthCodeExpiredError struct{}

func (e *AuthCodeNotFoundError) Error() string { return "auth code not found" }
func (e *AuthCodeUsedError) Error() string     { return "auth code already used" }
func (e *AuthCodeExpiredError) Error() string  { return "auth code expired" }

type TokenNotFoundError struct{}
type TokenExpiredError struct{}
type TokenInvalidClientError struct{}

func (e *TokenNotFoundError) Error() string { return "token not found" }
func (e *TokenExpiredError) Error() string  { return "token expired" }
func (e *TokenInvalidClientError) Error() string { return "invalid client" }
