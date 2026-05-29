package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

const (
	defaultListenAddr   = "127.0.0.1:8080"
	defaultLLMApexAddr  = "http://127.0.0.1:11434"
	sessionName         = "oauth2_proxy"
	stateCookieName     = "oauth2_state"
	codeCookieName      = "oauth2_code"
	redirectCookieName  = "oauth2_redirect"
	sessionTimeout     = 15 * time.Minute
	codeTimeout        = 2 * time.Minute
	maxRequestSize     = 10 << 20 // 10MB
)

var (
	allowedHosts = map[string]bool{
		"localhost":          true,
		"127.0.0.1":         true,
		"0.0.0.0":           true,
		"llm.example.com":   true,
	}
)

type Config struct {
	ListenAddr     string   `json:"listen_addr"`
	LLMApiAddr     string   `json:"llm_api_addr"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	RedirectURI    string   `json:"redirect_uri"`
	AuthServerURL  string   `json:"auth_server_url"`
	Scopes         []string `json:"scopes"`
	SkipAuth       bool     `json:"skip_auth"`
	TLSEnabled     bool     `json:"tls_enabled"`
	CertFile       string   `json:"cert_file"`
	KeyFile        string   `json:"key_file"`
	SessionMaxAge  int      `json:"session_max_age"`
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedPaths   []string `json:"allowed_paths"`
}

type State struct {
	ClientID    string
	RedirectURI string
	Nonce       string
	Code        string
	Expiry      time.Time
}

type Handler struct {
	config         Config
	store          sessions.Store
	stateCookie    *http.Cookie
	codeCookie     *http.Cookie
	redirectCookie *http.Cookie
	nonceCookie    *http.Cookie
	allowedOrigins string
}

type SessionData struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Nonce       string `json:"nonce"`
	Code        string `json:"code"`
	Expiry      int64  `json:"expiry"`
}

func main() {
	config := loadConfig()
	config = applyDefaults(config)

	handler := NewHandler(config)
	log.Printf("OAuth2 Proxy starting on %s", config.ListenAddr)

	if config.TLSEnabled {
		log.Fatal(http.ListenAndServeTLS(config.ListenAddr, config.CertFile, config.KeyFile, handler))
	}
	log.Fatal(http.ListenAndServe(config.ListenAddr, handler))
}

func loadConfig() Config {
	config := Config{
		ListenAddr:     defaultListenAddr,
		LLMApiAddr:     defaultLLMApexAddr,
		SessionMaxAge:  int(sessionTimeout.Seconds()),
	}

	if data, err := os.ReadFile(".oauth2config.json"); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("Warning: could not load config: %v", err)
		}
	}

	return config
}

func applyDefaults(config Config) Config {
	if config.ListenAddr == "" {
		config.ListenAddr = defaultListenAddr
	}
	if config.LLMApiAddr == "" {
		config.LLMApiAddr = defaultLLMApexAddr
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		log.Println("Warning: ClientID or ClientSecret not set - authorization disabled")
		config.SkipAuth = true
	}
	if config.RedirectURI == "" {
		config.RedirectURI = fmt.Sprintf("%s/callback", config.ListenAddr)
	}
	if config.AuthServerURL == "" {
		config.AuthServerURL = "http://localhost:3000"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}
	if config.SessionMaxAge == 0 {
		config.SessionMaxAge = int(sessionTimeout.Seconds())
	}
	return config
}

func NewHandler(config Config) *Handler {
	store := sessions.NewCookieStore([]byte("this-is-a-secret-key-for-cookies-min-32-bytes"))

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   config.SessionMaxAge,
		HttpOnly: true,
		Secure:   config.TLSEnabled,
		SameSite: http.SameSiteStrictMode,
	}

	return &Handler{
		config:        config,
		store:         store,
		stateCookie:   &http.Cookie{Name: stateCookieName, HttpOnly: false, SameSite: http.SameSiteLaxMode},
		codeCookie:    &http.Cookie{Name: codeCookieName, HttpOnly: false, SameSite: http.SameSiteLaxMode},
		redirectCookie: &http.Cookie{Name: redirectCookieName, HttpOnly: false, SameSite: http.SameSiteLaxMode},
		nonceCookie:   &http.Cookie{Name: "oauth2_nonce", HttpOnly: false, SameSite: http.SameSiteLaxMode},
		allowedOrigins: strings.Join(config.AllowedOrigins, ","),
	}
}

// ==================== HTTP Handlers ====================

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w, r)
		return
	}

	switch r.URL.Path {
	case "/health":
		h.handleHealth(w, r)
	case "/callback":
		h.handleCallback(w, r)
	case "/":
		h.handleIndex(w, r)
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.proxyLLMRequest(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if h.config.SkipAuth {
		proxyRequest(w, r, h.config.LLMApiAddr)
		return
	}

	if r.URL.Query().Get("authorize") == "true" {
		if err := h.forceAuth(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Proxy for LLM</title>
    <meta http-equiv="refresh" content="5;url=/authorize">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { text-align: center; padding: 40px; background: #f9f9f9; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        .status { color: #666; margin-top: 20px; }
        .btn { display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 OAuth2 Proxy</h1>
        <p>Redirecting you to authentication...</p>
        <a href="/authorize" class="btn">Authorize Now</a>
        <p class="status">Or you will be redirected automatically</p>
    </div>
</body>
</html>`))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","timestamp":"`+time.Now().Format(time.RFC3339)+`"}`))
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	_ = r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Authorization code missing", http.StatusBadRequest)
		return
	}

	// Get state from cookie
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Error(w, "State cookie missing", http.StatusBadRequest)
		return
	}

	// Verify state
	storedState, err := parseState(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if storedState.Code != code {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.exchangeCodeForToken(code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Create session
	session, err := h.store.Get(r, sessionName)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	sessionData := SessionData{
		ClientID:    storedState.ClientID,
		RedirectURI: storedState.RedirectURI,
		Expiry:      time.Now().Unix(),
	}
	if err := json.Unmarshal([]byte(storedState.Nonce), &sessionData); err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values["data"] = SessionData{
		ClientID:    storedState.ClientID,
		RedirectURI: storedState.RedirectURI,
		Expiry:      time.Now().Unix(),
		Code:        code,
	}

	h.store.Save(r, w, session)

	// Set cookies for proxying
	cookies := h.buildCookies(storedState.ClientID, token.AccessToken, token.ExpiresAt, storedState.Nonce)
	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
	}

	// Clear state cookies
	http.SetCookie(w, &http.Cookie{
		Name:   stateCookieName,
		Value:  "",
		Expires: time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:   codeCookieName,
		Value:  "",
		Expires: time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:   redirectCookieName,
		Value:  "",
		Expires: time.Unix(0, 0),
	})

	http.Redirect(w, r, storedState.RedirectURI, http.StatusSeeOther)
}

func (h *Handler) handleCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if h.allowedOrigins != "" && !contains(h.allowedOrigins, origin) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
	}
}

// ==================== Authorization Flow ====================

func (h *Handler) forceAuth(w http.ResponseWriter, r *http.Request) error {
	session, err := h.store.Get(r, sessionName)
	if err != nil {
		return fmt.Errorf("session error: %v", err)
	}

	forceFlag, forceExists := session.Values["force"]
	if !forceExists || forceFlag != true {
		// No force flag, do normal redirect
		http.Redirect(w, r, "/authorize", http.StatusSeeOther)
		return nil
	}

	// Generate state
	state, err := h.generateState(w)
	if err != nil {
		return err
	}

	// Exchange code immediately for testing
	code, err := h.exchangeCodeForToken(state.Code)
	if err != nil {
		return err
	}

	// Set auth cookies
	cookies := h.buildCookies(state.ClientID, code.AccessToken, code.ExpiresAt, state.Nonce)
	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
	}

	// Clear force flag
	session.Values["force"] = false
	h.store.Save(r, w, session)

	return nil
}

func generateNonce() string {
	b, _ := generateRandomString(16)
	return b
}

func (h *Handler) generateState(w http.ResponseWriter) (*State, error) {
	// Generate code challenge
	codeVerifier, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	codeChallenge, err := generateCodeChallenge(codeVerifier)
	if err != nil {
		return nil, err
	}

	// Create state
	state := &State{
		ClientID:    "local-client",
		RedirectURI: h.config.RedirectURI,
		Nonce:       generateNonce(),
		Code:        fmt.Sprintf("%s|%s", codeVerifier, codeChallenge),
		Expiry:      time.Now().Add(codeTimeout),
	}

	// Store in cookie
	cookie := &http.Cookie{
		Name:     stateCookieName,
		Value:    state.Code,
		Expires:  time.Now().Add(codeTimeout),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	return state, nil
}

type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
	ExpiresAt    time.Time
}

func (h *Handler) exchangeCodeForToken(code string) (*Token, error) {
	// For demo: simulate token response
	// In production, this would be a real OAuth2 token exchange
	accB, _ := generateRandomString(16)
	refB, _ := generateRandomString(32)
	accessToken := "demo-access-token-" + accB
	refreshToken := "demo-refresh-token-" + refB
	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil
}

// exchangeCodeForTokenWithValidation - validates code format before exchange
func (h *Handler) exchangeCodeForTokenWithValidation(code string) (*Token, error) {
	// Parse code format for PKCE validation
	parts := strings.SplitN(code, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid code format: expected 'verifier|challenge'")
	}

	// For demo: simulate token response
	// In production, this would be a real OAuth2 token exchange
	accB, _ := generateRandomString(16)
	refB, _ := generateRandomString(32)
	accessToken := "demo-access-token-" + accB
	refreshToken := "demo-refresh-token-" + refB
	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil
}

// ==================== Proxying ====================

func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Proxy error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = copyResponse(w, resp.Body)
	if err != nil {
		http.Error(w, "Response copy error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) proxyLLMRequest(w http.ResponseWriter, r *http.Request) {
	if h.config.SkipAuth {
		proxyRequest(w, r, h.config.LLMApiAddr)
		return
	}

	// Check auth cookie
	authCookie, err := r.Cookie(codeCookieName)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if authCookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Target URL - prepend LLMApiAddr and strip /api/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	if path == "" {
		path = "/"
	}

	// Handle both URLs with and without scheme
	var baseURL string
	if strings.HasPrefix(h.config.LLMApiAddr, "http://") || strings.HasPrefix(h.config.LLMApiAddr, "https://") {
		baseURL = strings.TrimPrefix(h.config.LLMApiAddr, "http://")
		if baseURL == "" {
			baseURL = strings.TrimPrefix(h.config.LLMApiAddr, "https://")
		}
	} else {
		baseURL = h.config.LLMApiAddr
	}
	targetURL := baseURL + "/" + path

	log.Printf("DEBUG: targetURL=%s, path=%s, LLMApiAddr=%s", targetURL, path, h.config.LLMApiAddr)

	// Add authorization header
	req, err := http.NewRequest(r.Method, "http://"+targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+authCookie.Value)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", r.Host)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Proxy error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	_, err = copyResponse(w, resp.Body)
	if err != nil {
		http.Error(w, "Response copy error", http.StatusInternalServerError)
		return
	}
}

// ==================== Utility Functions ====================

func generateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateCodeChallenge(codeVerifier string) (string, error) {
	h := sha256.Sum256([]byte(codeVerifier))
	return base64.URLEncoding.EncodeToString(h[:]), nil
}

func parseState(state string) (*State, error) {
	parts := strings.SplitN(state, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state format")
	}

	codeVerifier := parts[0]
	codeChallenge := parts[1]

	nonceB, _ := generateRandomString(16)
	nonce := nonceB
	return &State{
		ClientID:    "local-client",
		RedirectURI: "http://127.0.0.1:8080/callback",
		Nonce:       nonce,
		Code:        fmt.Sprintf("%s|%s", codeVerifier, codeChallenge),
		Expiry:      time.Now().Add(codeTimeout),
	}, nil
}

func init() {
	_ = sha256.Sum256
}

func copyResponse(w http.ResponseWriter, r io.Reader) (int64, error) {
	return io.Copy(w, r)
}

func (h *Handler) buildCookies(clientID, accessToken string, expiresAt time.Time, nonce string) []*http.Cookie {
	// Access token cookie
	accessCookie := &http.Cookie{
		Name:     codeCookieName,
		Value:    accessToken,
		Expires:  expiresAt,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}

	// Session cookie
	sessionData := map[string]interface{}{
		"client_id": clientID,
		"nonce":     nonce,
	}
	sessionJSON, _ := json.Marshal(sessionData)
	sessionCookie := &http.Cookie{
		Name:     sessionName,
		Value:    string(sessionJSON),
		MaxAge:   h.config.SessionMaxAge,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.TLSEnabled,
		SameSite: http.SameSiteStrictMode,
	}

	return []*http.Cookie{accessCookie, sessionCookie}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func init() {
	_ = sha256.Sum256
}
