package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"oauth2-proxy/pkg/oauth2/store"
)

const (
	DefaultCodeChallengeMethod = "S256"
	DefaultTokenType           = "Bearer"
	DefaultTokenLifetime       = 3600 // 1 hour
	RefreshTokenLifetime       = 86400 // 24 hours
)

type Server struct {
	Store            store.Store
	AuthorizationURL string
	CallbackURL      string
	ClientID         string
	ClientSecret     string
	CallbackHandler  http.HandlerFunc
}

func NewServer(
	store store.Store,
	authorizationURL, callbackURL, clientID, clientSecret string,
) (*Server, error) {
	if authorizationURL == "" {
		authorizationURL = "http://localhost:8081/oauth2/authorize"
	}
	if callbackURL == "" {
		callbackURL = "http://localhost:8080/oauth2/callback"
	}
	if clientID == "" {
		clientID = "proxy-client"
	}
	if clientSecret == "" {
		clientSecret = "proxy-secret"
	}

	return &Server{
		Store:           store,
		AuthorizationURL: authorizationURL,
		CallbackURL:     callbackURL,
		ClientID:        clientID,
		ClientSecret:    clientSecret,
	}, nil
}

func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// Parse request parameters
	redirectURI := r.FormValue("redirect_uri")
	responseType := r.FormValue("response_type")
	scope := r.FormValue("scope")

	// Validate response type
	if responseType != "code" {
		http.Error(w, "Unsupported response_type", http.StatusBadRequest)
		return
	}

	// Validate redirect URI
	if redirectURI == "" {
		http.Error(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}

	// Get or create user
	user, err := s.Store.GetOrCreateUser("proxy_user", "Proxy User", "proxy@example.com")
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create tokens
	accessToken := fmt.Sprintf("%s-access-token-%d", s.ClientID, time.Now().Unix())
	refreshToken := fmt.Sprintf("%s-refresh-token-%d", s.ClientID, time.Now().Unix())

	// Store tokens
	_, err = s.Store.CreateToken(user.ID, accessToken, refreshToken, DefaultTokenType, scope, DefaultTokenLifetime)
	if err != nil {
		log.Printf("Failed to store tokens: %v", err)
	}

	// For demo, redirect immediately
	http.Redirect(w, r, redirectURI, http.StatusFound)
}

func (s *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	authCode := r.FormValue("code")
	state := r.FormValue("state")

	if authCode == "" {
		http.Error(w, "Authorization code is required", http.StatusBadRequest)
		return
	}

	// Validate state
	if state == "" || state != "state" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get or create user
	user, err := s.Store.GetOrCreateUser("proxy_user", "Proxy User", "proxy@example.com")
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create tokens
	tokens := &TokenResponse{
		AccessToken: fmt.Sprintf("%s-access-token-%d", s.ClientID, time.Now().Unix()),
		RefreshToken: fmt.Sprintf("%s-refresh-token-%d", s.ClientID, time.Now().Unix()),
		TokenType:   DefaultTokenType,
		ExpiresIn:   DefaultTokenLifetime,
		Scope:       "read",
	}

	// Store tokens
	if _, err := s.Store.CreateToken(user.ID, tokens.AccessToken, tokens.RefreshToken, tokens.TokenType, tokens.Scope, tokens.ExpiresIn); err != nil {
		log.Printf("Failed to store tokens: %v", err)
	}

	// Create session
	sessionStore := store.NewInMemorySessionStore()
	session := &store.Session{
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}

	sessionStore.Store("oauth2_session", session)

	// Redirect to callback URL
	callbackURL := s.CallbackURL
	if r.URL.RawQuery != "" {
		callbackURL += "&" + r.URL.RawQuery
	}
	http.Redirect(w, r, callbackURL, http.StatusFound)
}

func (s *Server) generateStateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *Server) generateCodeChallenge(_ string) (string, string, string) {
	// Generate code verifier (43-128 characters)
	codeVerifier := s.generateCodeVerifier()

	// Generate code challenge
	codeChallenge := s.codeChallengeFromVerifier(codeVerifier)
	codeChallengeMethod := DefaultCodeChallengeMethod

	return codeVerifier, codeChallenge, codeChallengeMethod
}

func (s *Server) generateCodeVerifier() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:43]
}

func (s *Server) codeChallengeFromVerifier(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	encoded := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return encoded[:43]
}

func (s *Server) GetStore() store.SessionStore {
	return store.NewInMemorySessionStore()
}

// AuthRequest represents an authorization request
type AuthRequest struct {
	StateToken        string
	CodeVerifier      string
	CodeChallenge     string
	CodeChallengeMethod string
	RedirectURI       string
	Scope             string
}

// TokenResponse represents OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}
