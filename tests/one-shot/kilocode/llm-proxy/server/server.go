package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-proxy/config"
	"llm-proxy/models"
	"llm-proxy/storage"
)

type Server struct {
	config *config.Config
	store  storage.Store
}

func NewServer(cfg *config.Config, store storage.Store) *Server {
	return &Server{
		config: cfg,
		store:  store,
	}
}

func (s *Server) Run() error {
	addr := s.config.ListenAddr
	if s.config.Certificate != "" && s.config.Key != "" {
		return s.runTLS(addr)
	}
	return s.runHTTP(addr)
}

func (s *Server) runHTTP(addr string) error {
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r, s.config, s.store)
	})

	server.Handler = handler

	return server.ListenAndServe()
}

func (s *Server) runTLS(addr string) error {
	cert, err := tls.LoadX509KeyPair(s.config.Certificate, s.config.Key)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r, s.config, s.store)
	})

	server.Handler = handler

	return server.ListenAndServeTLS("", "")
}

func (s *Server) Cleanup() {
	s.store.CleanExpired()
}

func next(w http.ResponseWriter, r *http.Request, cfg *config.Config, store storage.Store) {
	switch r.URL.Path {
	case "/oauth/authorize":
		authorize(w, r, cfg, store)
	case "/oauth/token":
		token(w, r, cfg, store)
	case "/oauth/userinfo":
		userInfo(w, r, store)
	case "/":
		health(w, r)
	default:
		proxyRequest(w, r, cfg, store)
	}
}

func authorize(w http.ResponseWriter, r *http.Request, cfg *config.Config, store storage.Store) {
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	scopes := r.FormValue("scope")
	state := r.FormValue("state")

	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	if redirectURI == "" {
		http.Error(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}

	code, err := generateAuthCode(store, clientID, scopes, redirectURI, state)
	if err != nil {
		http.Error(w, "failed to generate auth code", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state)
	http.Redirect(w, r, url, http.StatusFound)
}

func token(w http.ResponseWriter, r *http.Request, cfg *config.Config, store storage.Store) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	grantType := r.FormValue("grant_type")

	if code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	if clientID == "" || clientSecret == "" {
		http.Error(w, "client_id and client_secret are required", http.StatusBadRequest)
		return
	}

	if grantType != "authorization_code" {
		http.Error(w, "grant_type must be 'authorization_code'", http.StatusBadRequest)
		return
	}

	authCode, err := store.GetAuthCode(code)
	if err != nil {
		http.Error(w, "invalid auth code", http.StatusBadRequest)
		return
	}

	if authCode.ClientID != clientID {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}

	tok, err := generateToken(clientID, authCode)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	if err := store.SaveToken(tok); err != nil {
		http.Error(w, "failed to save token", http.StatusInternalServerError)
		return
	}

	if err := store.RemoveAuthCode(code); err != nil {
		// Log error but don't fail
	}

	response := &models.TokenResponse{
		AccessToken:  tok.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: tok.RefreshToken,
		Scope:        authCode.Scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func userInfo(w http.ResponseWriter, r *http.Request, store storage.Store) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		return
	}

	claims, err := parseJWTToken(authHeader)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userInfo := &models.UserInfo{
		ID:          claims.ID,
		Email:       "",
		DisplayName: claims.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func proxyRequest(w http.ResponseWriter, r *http.Request, cfg *config.Config, store storage.Store) {
	paths := []string{
		"/api/generate",
		"/api/chat",
		"/v1/chat/completions",
		"/v1/completions",
	}

	isProxyPath := false
	for _, path := range paths {
		if r.URL.Path == path || r.URL.Path == "/"+path {
			isProxyPath = true
			break
		}
	}

	if !isProxyPath {
		http.NotFound(w, r)
		return
	}

	token, err := getClientFromToken(r, store)
	if err != nil {
		if err.Error() == "invalid token" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
		} else {
			http.Error(w, "authorization required", http.StatusForbidden)
		}
		return
	}

	if token == nil {
		http.Error(w, "authorization required", http.StatusForbidden)
		return
	}

	targetURL := cfg.LLMAPIURL
	if targetURL == "" {
		targetURL = "http://localhost:11434/api/generate"
	}

	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	originalBody, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(originalBody))

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		req.Header[k] = v
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.WriteHeader(resp.StatusCode)
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.Write(body)
}

func getClientFromToken(r *http.Request, store storage.Store) (*models.OAuth2Token, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid token")
	}

	return store.GetToken(parts[1], "")
}

// Helpers
func generateAuthCode(store storage.Store, clientID, scopes, redirectURI, state string) (string, error) {
	code := randomString(16)

	authCode := &models.OAuth2AuthCode{
		Code:          code,
		ClientID:      clientID,
		Scopes:        scopes,
		RedirectURI:   redirectURI,
		Nonce:         "",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		Used:          false,
	}

	if err := store.SaveAuthCode(authCode); err != nil {
		return "", err
	}

	return code, nil
}

func generateToken(clientID string, authCode *models.OAuth2AuthCode) (*models.OAuth2Token, error) {
	code := randomString(16)

	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	token := &models.OAuth2Token{
		AccessToken:   code,
		RefreshToken:  randomString(32),
		ExpiresAt:     expiresAt,
		Scopes:        authCode.Scopes,
		Code:          code,
		ClientID:      clientID,
		Nonce:         "",
		CreatedAt:     now,
	}

	return token, nil
}

func parseJWTToken(authHeader string) (*models.UserInfo, error) {
	return &models.UserInfo{ID: "user-123"}, nil
}

type corsConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

func randomString(n int) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}
