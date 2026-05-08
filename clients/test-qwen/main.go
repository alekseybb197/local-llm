package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds proxy configuration
type Config struct {
	LLMHost       string
	LLMPort       int
	LLMPath       string
	AdminUser     string
	AdminPass     string
	ExposeMetrics bool
}

// Default config
var config = Config{
	LLMHost:     "localhost",
	LLMPort:     11434, // Ollama default
	LLMPath:     "/api/generate",
	AdminUser:   "admin",
	AdminPass:   "password",
	ExposeMetrics: false,
}

// Metrics
var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_requests_total",
			Help: "Total number of LLM proxy requests",
		},
		[]string{"status"},
	)
)

func initMetrics() {
	prometheus.MustRegister(requestCount)
}

// sessions handles OAuth2 session storage
type sessions struct {
	store *securecookie.CookieStore
}

func (s *sessions) SaveState(r http.ResponseWriter, state string) {
	cookie := &http.Cookie{
		Name:     "oauth2_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(r, cookie)
}

func (s *sessions) GetState(r *http.Request) (string, error) {
	cookie, err := r.Cookie("oauth2_state")
	if err != nil {
		return "", err
	}

	var state string
	if err := s.store.Decode("oauth2_state", []byte(cookie.Value), &state); err != nil {
		return "", err
	}
	return state, nil
}

// oauth2Token holds OAuth2 tokens
type oauth2Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time
}

// Session represents an authenticated session
type Session struct {
	User   string         `json:"user"`
	Expiry time.Time      `json:"expiry"`
	Token  oauth2Token    `json:"token"`
}

func (s *sessions) SaveSession(r http.ResponseWriter, session Session) error {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    s.encodeSession(session),
		Path:     "/",
		Expires:  session.Expiry,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(r, cookie)
	return nil
}

func (s *sessions) GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil, err
	}

	var session Session
	if err := s.store.Decode("session", []byte(cookie.Value), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// encodeSession encodes session to string
func (s *sessions) encodeSession(session Session) string {
	// Create a temporary struct to encode
	type sessionStruct struct {
		User   string    `json:"user"`
		Expiry string    `json:"expiry"`
		Token  oauth2Token `json:"token"`
	}
	temp := sessionStruct{
		User:   session.User,
		Expiry: session.Expiry.Format(time.RFC3339),
		Token:  session.Token,
	}
	data, _ := json.Marshal(temp)
	return base64.StdEncoding.EncodeToString(data)
}

// llmRequest represents a request to the LLM
type llmRequest struct {
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Context  []int                  `json:"context,omitempty"`
	System   string                 `json:"system,omitempty"`
	Format   string                 `json:"format,omitempty"`
	KeepAlive *time.Duration         `json:"keep_alive,omitempty"`
}

// llmResponse represents the LLM response
type llmResponse struct {
	Model       string      `json:"model"`
	CreatedAt   time.Time   `json:"created_at"`
	Done        bool        `json:"done"`
	TotalDuration time.Duration `json:"total_duration,omitempty"`
	LoadDuration time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount int     `json:"eval_count,omitempty"`
	EvalDuration time.Duration `json:"eval_duration,omitempty"`
	Response    string      `json:"response"`
	Error       string      `json:"error,omitempty"`
}

// llmGenerateResponse for streaming
type llmGenerateResponse struct {
	Model       string      `json:"model"`
	Done        bool        `json:"done"`
	TotalDuration time.Duration `json:"total_duration,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount int     `json:"eval_count,omitempty"`
	EvalDuration time.Duration `json:"eval_duration,omitempty"`
	Response    string      `json:"response"`
	DoneReason  string      `json:"done_reason,omitempty"`
}

// llmClient handles communication with the LLM
type llmClient struct {
	host     string
	port     int
	path     string
	token    string
	baseURL  string
}

func newLLMClient(host, port, token string) *llmClient {
	return &llmClient{
		host: host,
		port: port,
		token: token,
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
	}
}

// generate sends a request to the LLM
func (c *llmClient) generate(req llmRequest) (*llmResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL := c.baseURL + c.path
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var response llmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// stream sends a streaming request to the LLM
func (c *llmClient) stream(req llmRequest) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := c.baseURL + c.path
	reqURL = strings.Replace(reqURL, "/api/generate", "/api/generate?stream=true", 1)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp.Body, nil
}

// generateStateToken creates a random state token
func generateStateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	// Load config from environment
	if host := os.Getenv("LLM_HOST"); host != "" {
		config.LLMHost = host
	}
	if port := os.Getenv("LLM_PORT"); port != "" {
		var err error
		config.LLMPort, err = strconv.Atoi(port)
		if err != nil {
			log.Printf("Invalid LLM_PORT: %v, using default", err)
		}
	}
	if user := os.Getenv("ADMIN_USER"); user != "" {
		config.AdminUser = user
	}
	if pass := os.Getenv("ADMIN_PASS"); pass != "" {
		config.AdminPass = pass
	}
	if expose := os.Getenv("EXPOSE_METRICS"); expose == "true" {
		config.ExposeMetrics = true
	}

	initMetrics()

	// Initialize secure cookie
	cookiePassword := []byte("super-secret-key-for-encryption-change-this-in-production")
	secureCookie := securecookie.New(securecookie.GenerateRandomKey(32), cookiePassword)

	// Create session store
	sessionStore := sessions{
		store: secureCookie,
	}

	// Create router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// OAuth2 flow
	router.HandleFunc("/login", loginHandler).Methods("GET")
	router.HandleFunc("/oauth2/callback", callbackHandler).Methods("GET")

	// Protected API endpoint
	router.HandleFunc("/api/generate", generateHandler).Methods("POST")
	router.HandleFunc("/api/generate/stream", streamHandler).Methods("POST")

	// Serve metrics if enabled
	if config.ExposeMetrics {
		router.Handle("/metrics", promhttp.Handler())
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting LLM proxy on port %s", port)
	log.Printf("LLM endpoint: http://%s:%d%s", config.LLMHost, config.LLMPort, config.LLMPath)
	log.Printf("Admin user: %s", config.AdminUser)

	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// loginHandler handles OAuth2 login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirect := query.Get("redirect")
	if redirect == "" {
		redirect = "/api/generate"
	}

	// Generate state token
	stateToken, err := generateStateToken()
	if err != nil {
		http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}
	sessionStore.SaveState(w, stateToken)

	// Generate OAuth2 authorization URL
	authURL := fmt.Sprintf(
		"http://%s:%s/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		config.LLMHost,
		config.LLMPort,
		"llm-proxy-client",
		url.QueryEscape(redirect),
		stateToken,
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// callbackHandler handles OAuth2 callback
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	stateToken := query.Get("state")
	code := query.Get("code")

	// Verify state token
	session, err := sessionStore.GetState(r)
	if err != nil || session != stateToken {
		http.Error(w, "Invalid state token", http.StatusForbidden)
		return
	}

	// Exchange code for token
	accessToken := exchangeCodeForToken(code)

	// Create and store session
	sess := Session{
		User:   config.AdminUser,
		Expiry: time.Now().Add(24 * time.Hour),
		Token:  oauth2Token{AccessToken: accessToken},
	}

	if err := sessionStore.SaveSession(w, sess); err != nil {
		log.Printf("Failed to save session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Redirect to original redirect URL
	http.Redirect(w, r, r.URL.Query().Get("redirect"), http.StatusFound)
}

// exchangeCodeForToken exchanges authorization code for access token
func exchangeCodeForToken(code string) string {
	// TODO: Implement token exchange with LLM provider
	// This would typically call the LLM's OAuth2 token endpoint
	// Example: POST to /oauth/token with code
	return ""
}

// generateHandler handles LLM generation requests with OAuth2 authentication
func generateHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	vars := mux.Vars(r)
	user := vars["user"]

	// Verify OAuth2 session
	session, err := getSession(r)
	if err != nil {
		log.Printf("Invalid session: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify user
	if session.User != user {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get request body
	var req llmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Update metrics
	requestCount.WithLabelValues("in_progress").Inc()
	defer func() {
		status := "success"
		if r.URL.Query().Get("error") == "true" {
			status = "error"
		}
		requestCount.WithLabelValues(status).Inc()
	}()

	// Create LLM client
	client := newLLMClient(
		config.LLMHost,
		config.LLMPort,
		session.Token.AccessToken,
	)

	// Forward request to LLM
	response, err := client.generate(req)
	if err != nil {
		log.Printf("LLM request failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// streamHandler handles streaming LLM requests
func streamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]

	// Verify OAuth2 session
	session, err := getSession(r)
	if err != nil {
		log.Printf("Invalid session: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify user
	if session.User != user {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get request body
	var req llmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Set streaming headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Create LLM client
	client := newLLMClient(
		config.LLMHost,
		config.LLMPort,
		session.Token.AccessToken,
	)

	// Create streaming request
	stream, err := client.stream(req)
	if err != nil {
		log.Printf("LLM stream failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer stream.Close()

	// Handle streaming response
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err != nil || n == 0 {
				break
			}
			if _, err := w.Write(buf[:n]); err != nil {
				break
			}
		}
	}()

	<-done
}

// getSession retrieves and validates the OAuth2 session
func getSession(r *http.Request) (*Session, error) {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return nil, err
	}

	var session Session
	if err := sessionStore.GetSession(r, &session); err != nil {
		return nil, err
	}

	// Check if session has expired
	if time.Now().After(session.Expiry) {
		return nil, nil
	}

	return &session, nil
}

// urlQueryEscape escapes a URL query parameter
func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}
