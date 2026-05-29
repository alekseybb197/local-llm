package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ==================== Configuration ====================

const (
	defaultPort     = 8080
	jwtSecretKey    = "oauth2-proxy-secret-key-change-in-production"
	tokenExpiry     = time.Hour * 2
	stateSecretLen  = 32
)

var (
	errInvalidState      = errors.New("invalid state parameter")
	errInvalidCode       = errors.New("invalid authorization code")
	errInvalidToken      = errors.New("invalid access token")
	errInvalidGrantType  = errors.New("invalid grant type")
	errInvalidClient     = errors.New("invalid client credentials")
	errTokenExpired      = errors.New("token has expired")
	errInvalidScope      = errors.New("invalid scope")
	errMissingParam      = errors.New("missing required parameter")
	errMalformedURL      = errors.New("malformed URL")
	errInternal          = errors.New("internal server error")
)

// ==================== Types ====================

type ServerConfig struct {
	Host         string
	Port         int
	RedirectURI  string
	CertFile     string
	KeyFile      string
	AuthorizationURL string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

type OAuth2Config struct {
	AuthorizationURL string
	TokenURL         string
	UserInfoURL      string
	ClientID         string
	ClientSecret     string
	Scopes           []string
	RedirectURI      string
	StateSecret      string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
}

type AuthState struct {
	CodeChallenge             string
	CodeChallengeMethod       string
	ClientID                  string
	Scopes                    []string
	RedirectURI               string
	State                     string
	CodeChallengeMethodHeader string
}

// ==================== Server ====================

type OAuth2ProxyServer struct {
	config        ServerConfig
	oauth2Config  OAuth2Config
	tokens        map[string]*TokenData
	tokenMutex    sync.RWMutex
	codeToState   map[string]*AuthState
	codeMutex     sync.RWMutex
	mux           *http.ServeMux
}

type TokenData struct {
	AccessToken  string
	RefreshToken string
	Scopes       []string
	UserInfo     UserInfo
	ExpiresAt    time.Time
	IssuedAt     time.Time
	Claims       jwt.RegisteredClaims
}

type CodeData struct {
	State       *AuthState
	ExpiresAt   time.Time
	Code        string
	Hash        string
}

// ==================== Helpers ====================

func generateState() (string, error) {
	bytes := make([]byte, stateSecretLen)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateCodeChallenge(codeVerifier string) string {
	challenge := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(challenge[:])
}

func generateJWTClaims(clientID, userID string, scopes []string, expiresAt time.Time) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:       userID,
		IssuedAt:      jwt.NewNumericDate(time.Now()),
		ExpiresAt:     jwt.NewNumericDate(expiresAt),
		Issuer:        "oauth2-proxy",
		NotBefore:     jwt.NewNumericDate(time.Now()),
		Audience:      jwt.ClaimStrings{"api"},
	}
}

func generateJWT(claims *jwt.RegisteredClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecretKey))
	return tokenString
}

// ==================== Authentication ====================

func getOAuth2Config(r *http.Request) (*OAuth2Config, error) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")

	if clientID == "" {
		return nil, errMissingParam
	}
	if redirectURI == "" {
		return nil, errMissingParam
	}

	return &OAuth2Config{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		Scopes:        []string{"read", "write"},
		AuthorizationURL: "https://example.com/oauth/authorize",
		TokenURL:        "https://example.com/oauth/token",
		UserInfoURL:     "https://example.com/oauth/userinfo",
	}, nil
}

func (s *OAuth2ProxyServer) authHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	oauth2Config, err := getOAuth2Config(r)
	if err != nil {
		log.Printf("Error getting OAuth2 config: %v", err)
		http.Error(w, "Invalid OAuth2 configuration", http.StatusBadRequest)
		return
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s",
		oauth2Config.AuthorizationURL,
		url.QueryEscape(oauth2Config.ClientID),
		url.QueryEscape(oauth2Config.RedirectURI),
		url.QueryEscape(strings.Join(oauth2Config.Scopes, " ")),
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Authorization</title>
</head>
<body>
    <h1>Authorization Required</h1>
    <p>You will be redirected to the authorization server...</p>
    <script>
        window.location.href = '` + authURL + `';
    </script>
    <a href="javascript:window.location.href='` + authURL + `'">Click here if not redirected</a>
</body>
</html>`
	w.Write([]byte(html))
}

func (s *OAuth2ProxyServer) callbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	if state != "" {
		if err := validateState(state); err != nil {
			log.Printf("Invalid state: %v", err)
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = s.oauth2Config.RedirectURI
	}

	http.Redirect(w, r, redirectURI, http.StatusFound)
}

func validateState(state string) error {
	if state == "" {
		return nil
	}
	_, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return err
	}
	return nil
}

// ==================== Token Endpoint ====================

func (s *OAuth2ProxyServer) tokenEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	switch req.GrantType {
	case "authorization_code":
		s.handleAuthorizationCodeGrant(w, r, &req)
	case "refresh_token":
		s.handleRefreshTokenGrant(w, r, &req)
	default:
		http.Error(w, fmt.Sprintf("Unsupported grant type: %s", req.GrantType), http.StatusBadRequest)
	}
}

func (s *OAuth2ProxyServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request, req *TokenRequest) {
	code := req.Code
	redirectURI := req.RedirectURI

	if code == "" {
		http.Error(w, errMissingParam.Error(), http.StatusBadRequest)
		return
	}

	if redirectURI == "" {
		http.Error(w, errMissingParam.Error(), http.StatusBadRequest)
		return
	}

	if err := validateRedirectURI(redirectURI); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tokenData := generateTokenData(
		req.ClientID,
		"mock-user-id",
		[]string{"read", "write"},
		time.Now().Add(time.Hour*2),
	)

	s.saveToken(tokenData.AccessToken, tokenData)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Expose-Headers", "Access-Token, TokenType, Expires-In, Refresh-Token, Scope")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  tokenData.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: tokenData.RefreshToken,
		Scope:        strings.Join(tokenData.Scopes, " "),
	})
}

func (s *OAuth2ProxyServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request, req *TokenRequest) {
	refreshToken := req.RefreshToken

	if refreshToken == "" {
		http.Error(w, errMissingParam.Error(), http.StatusBadRequest)
		return
	}

	tokenData := generateTokenData(
		req.ClientID,
		"mock-user-id",
		[]string{"read", "write"},
		time.Now().Add(time.Hour*2),
	)

	s.saveToken(tokenData.AccessToken, tokenData)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Expose-Headers", "Access-Token, TokenType, Expires-In, Refresh-Token, Scope")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  tokenData.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    7200,
		RefreshToken: tokenData.RefreshToken,
		Scope:        strings.Join(tokenData.Scopes, " "),
	})
}

func validateRedirectURI(redirectURI string) error {
	if redirectURI == "" {
		return errMissingParam
	}

	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return errMalformedURL
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errMalformedURL
	}

	return nil
}

// ==================== Token Validation ====================

func (s *OAuth2ProxyServer) saveToken(token string, tokenData *TokenData) {
	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()
	s.tokens[token] = tokenData
}

func (s *OAuth2ProxyServer) validateToken(token string) (*TokenData, error) {
	s.tokenMutex.RLock()
	defer s.tokenMutex.RUnlock()

	data, exists := s.tokens[token]
	if !exists {
		return nil, errInvalidToken
	}

	if time.Now().After(data.ExpiresAt) {
		return nil, errTokenExpired
	}

	return data, nil
}

func (s *OAuth2ProxyServer) validateJWT(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, errInvalidToken
	}

	return claims, nil
}

func (s *OAuth2ProxyServer) validateAuthorizationHeader(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if parts[0] != "Bearer" {
		return "", fmt.Errorf("unsupported authorization type")
	}

	return parts[1], nil
}

// ==================== UserInfo Endpoint ====================

func (s *OAuth2ProxyServer) userInfoEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	token, err := s.validateAuthorizationHeader(authHeader)
	if err != nil {
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenData, err := s.validateToken(token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenData.UserInfo)
}

// ==================== LLM Proxy ====================

type LLMResponse struct {
	ID       string   `json:"id"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Model    string   `json:"model"`
	Choices  []Choice `json:"choices"`
	Usage    Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (s *OAuth2ProxyServer) llmEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	token, err := s.validateAuthorizationHeader(authHeader)
	if err != nil {
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	_, err = s.validateToken(token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	proxyURL := "http://localhost:11434/v1/chat/completions"

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		DisableCompression:    true,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	client := &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}

	resp, err := client.Post(proxyURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, "Error forwarding request to LLM", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ==================== Routes ====================

func (s *OAuth2ProxyServer) setupRoutes() {
	s.mux.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) {
		s.authHandler(w, r)
	})
	s.mux.HandleFunc("/callback/", func(w http.ResponseWriter, r *http.Request) {
		s.callbackHandler(w, r)
	})
	s.mux.HandleFunc("/oauth/token/", func(w http.ResponseWriter, r *http.Request) {
		s.tokenEndpoint(w, r)
	})
	s.mux.HandleFunc("/oauth/userinfo/", func(w http.ResponseWriter, r *http.Request) {
		s.userInfoEndpoint(w, r)
	})
	s.mux.HandleFunc("/v1/chat/completions/", func(w http.ResponseWriter, r *http.Request) {
		s.llmEndpoint(w, r)
	})
}

// ==================== Server Methods ====================

func (s *OAuth2ProxyServer) Start() error {
	if s.config.CertFile != "" && s.config.KeyFile != "" {
		return http.ListenAndServeTLS(
			fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
			s.config.CertFile,
			s.config.KeyFile,
			s.mux,
		)
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%d", s.config.Host, s.config.Port), s.mux)
}

func (s *OAuth2ProxyServer) Stop() error {
	return nil
}

// ==================== Main ====================

func main() {
	config := loadConfig()
	server := NewOAuth2ProxyServer(config)

	log.Printf("OAuth2 Proxy Server starting on %s:%d", config.Host, config.Port)
	log.Println("Endpoints:")
	log.Println("  POST   /oauth/token          - Token endpoint")
	log.Println("  GET    /oauth/userinfo       - User info endpoint")
	log.Println("  GET    /auth                 - Authorization page")
	log.Println("  GET    /callback             - OAuth callback")
	log.Println("  POST   /v1/chat/completions  - LLM proxy endpoint")

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ==================== Config ====================

func loadConfig() ServerConfig {
	config := ServerConfig{
		Host:         "0.0.0.0",
		Port:         defaultPort,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/callback", defaultPort),
		ClientID:     "default-client-id",
		ClientSecret: "default-client-secret",
		Scopes:       []string{"read", "write"},
	}

	configFile := "config.json"
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("Warning: Failed to load config: %v, using defaults", err)
		}
	}

	return config
}

func NewOAuth2ProxyServer(config ServerConfig) *OAuth2ProxyServer {
	mux := http.NewServeMux()
	serverConfig := OAuth2Config{
		ClientID:         config.ClientID,
		ClientSecret:     config.ClientSecret,
		Scopes:           config.Scopes,
		RedirectURI:      config.RedirectURI,
		AuthorizationURL: config.AuthorizationURL,
		TokenURL:         config.TokenURL,
		UserInfoURL:      "https://example.com/oauth/userinfo",
		StateSecret:      "default-state-secret",
	}
	return &OAuth2ProxyServer{
		config:        config,
		oauth2Config:  serverConfig,
		tokens:        make(map[string]*TokenData),
		codeToState:   make(map[string]*AuthState),
		mux:           mux,
	}
}

func generateTokenData(clientID, userID string, scopes []string, expiresAt time.Time) *TokenData {
	claims := generateJWTClaims(clientID, userID, scopes, expiresAt)
	return &TokenData{
		AccessToken:  generateJWT(&claims),
		RefreshToken: uuid.New().String(),
		Scopes:       scopes,
		UserInfo: UserInfo{
			ID: userID,
		},
		ExpiresAt: expiresAt,
		IssuedAt:  time.Now(),
		Claims:    claims,
	}
}
