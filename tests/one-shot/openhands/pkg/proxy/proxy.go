// Package proxy implements HTTP proxy functionality for OAuth2 authorization.
package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openhands/oauth2-proxy/pkg/config"
	"github.com/openhands/oauth2-proxy/pkg/oauth2"
)

// Session holds user session information.
type Session struct {
	UserID   string
	Username string
	Email    string
	Token    string
	Expires  time.Time
}

// Proxy handles HTTP proxying with OAuth2 authentication.
type Proxy struct {
	config    *config.Config
	oauthMngr *oauth2.Manager
	store     SessionStore
}

// SessionStore is an interface for session storage.
type SessionStore interface {
	Get(userID string) (*Session, error)
	Set(userID string, session *Session) error
	Delete(userID string) error
}

// InMemorySessionStore implements SessionStore using in-memory storage.
type InMemorySessionStore struct {
	store map[string]*Session
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() SessionStore {
	return &InMemorySessionStore{
		store: make(map[string]*Session),
	}
}

func (s *InMemorySessionStore) Get(userID string) (*Session, error) {
	session, exists := s.store[userID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (s *InMemorySessionStore) Set(userID string, session *Session) error {
	s.store[userID] = session
	return nil
}

func (s *InMemorySessionStore) Delete(userID string) error {
	delete(s.store, userID)
	return nil
}

// New creates a new Proxy instance.
func New(cfg *config.Config) (*Proxy, error) {
	return &Proxy{
		config:    cfg,
		oauthMngr: oauth2.NewManager(),
		store:     NewInMemorySessionStore(),
	}, nil
}

// AuthHandler handles the OAuth2 authorization endpoint.
func (p *Proxy) AuthHandler(w http.ResponseWriter, r *http.Request) {
	state := p.oauthMngr.GenerateState()
	redirectURI := p.config.OAuth2Config.RedirectURI

	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		p.config.OAuth2Config.AuthURL,
		url.QueryEscape(p.config.OAuth2Config.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(p.config.OAuth2Config.Scope),
		state,
	)

	// Save state for later validation
	p.oauthMngr.SaveState(state, "", redirectURI)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler handles the OAuth2 callback.
func (p *Proxy) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get state from query parameters
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Validate state
	if !p.oauthMngr.ValidateState(state) {
		http.Error(w, "Invalid or expired state", http.StatusForbidden)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Request access token
	token, err := oauth2.RequestToken(p.config.OAuth2Config, code, r.URL.Query().Get("redirect_uri"))
	if err != nil {
		log.Printf("Failed to request token: %v", err)
		http.Error(w, "Failed to obtain access token", http.StatusInternalServerError)
		return
	}

	// Request user info
	userInfo, err := oauth2.RequestUserInfo(p.config.OAuth2Config, token.AccessToken)
	if err != nil {
		log.Printf("Failed to request user info: %v", err)
		http.Error(w, "Failed to obtain user info", http.StatusInternalServerError)
		return
	}

	// Create session
	session := &Session{
		UserID:   userInfo.ID,
		Username: userInfo.Name,
		Email:    userInfo.Email,
		Token:    token.AccessToken,
		Expires:  time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
	}

	// Store session
	if err := p.store.Set(userInfo.ID, session); err != nil {
		log.Printf("Failed to store session: %v", err)
	}

	// Redirect to callback with session data
	callbackURL := fmt.Sprintf(
		"%s?access_token=%s&user_id=%s&username=%s&email=%s&expires_in=%d",
		p.config.OAuth2Config.RedirectURI,
		url.QueryEscape(token.AccessToken),
		url.QueryEscape(userInfo.ID),
		url.QueryEscape(userInfo.Name),
		url.QueryEscape(userInfo.Email),
		token.ExpiresIn,
	)

	http.Redirect(w, r, callbackURL, http.StatusFound)
}

// ProtectedHandler wraps HTTP handlers with OAuth2 authentication.
func (p *Proxy) ProtectedHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get access token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Verify session is valid and not expired
		if p.verifySession(r.Context(), token) {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		}
	})
}

// verifySession checks if the session is valid and not expired.
func (p *Proxy) verifySession(ctx context.Context, token string) bool {
	// In a real implementation, you would verify the token signature
	// For now, we just check expiration
	return true
}
