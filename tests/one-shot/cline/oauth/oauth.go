package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

type OAuthServer struct {
	database    *db.Database
	config      *config.OAuthConfig
	httpClient  *http.Client
}

func NewOAuthServer(database *db.Database, oauthConfig *config.OAuthConfig) *OAuthServer {
	return &OAuthServer{
		database:   database,
		config:     oauthConfig,
		httpClient:  &http.Client{Timeout: oauthConfig.AuthorizationTimeout},
	}
}

func (s *OAuthServer) GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *OAuthServer) GenerateCodeChallenge(codeVerifier string, method string) (string, error) {
	var hash []byte
	switch method {
	case "S256":
		// Convert [32]byte to []byte by copying bytes
		hash = make([]byte, 32)
		for i := range hash {
			hash[i] = sha256.Sum256([]byte(codeVerifier))[i]
		}
	default:
		hash = []byte(codeVerifier)
	}
	return base64.RawURLEncoding.EncodeToString(hash), nil
}

func (s *OAuthServer) Authorize(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := url.Parse(s.config.RedirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}

	authURL := redirectURL.JoinPath("oauth/authorize")
	query := authURL.Query()

	state := r.FormValue("state")
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	query.Set("client_id", s.config.ClientID)
	query.Set("redirect_uri", s.config.RedirectURI)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(s.config.Scopes, " "))
	query.Set("state", state)
	query.Set("code_challenge", r.FormValue("code_challenge"))
	query.Set("code_challenge_method", r.FormValue("code_challenge_method"))

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (s *OAuthServer) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Store the authorization code as a token for later exchange
	token := &models.Token{
		AccessToken:  code,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(60 * time.Second), // Codes expire quickly
		RefreshToken: "",
		Scopes:       strings.Join(s.config.Scopes, " "),
		Subject:      "user",
		ClientID:     s.config.ClientID,
		CreatedAt:    time.Now(),
	}

	if err := s.database.CreateToken(token); err != nil {
		log.Printf("Failed to create token: %v", err)
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, s.config.RedirectURI, http.StatusFound)
}

func (s *OAuthServer) GetToken(ctx context.Context, code string) (*models.Token, error) {
	// Return a basic token - actual exchange happens in TokenExchange
	return &models.Token{
		AccessToken: code,
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(3600 * time.Second),
	}, nil
}

func (s *OAuthServer) RefreshToken(ctx context.Context, refreshToken string) (*models.Token, error) {
	// Placeholder - in production this would call the token endpoint
	return &models.Token{
		AccessToken: "refreshed_access_token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(3600 * time.Second),
	}, nil
}

func (s *OAuthServer) TokenExchange(ctx context.Context, clientID string, code string) (*models.Token, error) {
	// Exchange authorization code for token
	token := &models.Token{
		AccessToken: code,
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(3600 * time.Second),
	}

	// In production, this would make a POST request to /oauth/token
	// For now, just return the code as the access token
	return token, nil
}

func (s *OAuthServer) ExchangeCode(ctx context.Context, code string, redirectURL string) (*models.Token, error) {
	token := &models.Token{
		AccessToken: code,
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(3600 * time.Second),
	}

	// In production, this would make a POST request to token endpoint
	// For now, return the token with metadata
	return token, nil
}

func (s *OAuthServer) WithTx() (*sql.Tx, error) {
	return s.database.WithTx()
}
