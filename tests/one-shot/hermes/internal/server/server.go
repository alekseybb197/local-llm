package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"golang.org/x/oauth2"

	hermes "hermes/internal/store"
	hermesconfig "hermes/config"
)

// Server represents the HTTP server
type Server struct {
	config *hermesconfig.Config
	router *chi.Mux
	store  hermes.Store
}

// NewServer creates a new HTTP server
func NewServer(cfg *hermesconfig.Config, store hermes.Store) (*Server, error) {
	s := &Server{
		config: cfg,
		store:  store,
	}

	// Create router
	s.router = chi.NewMux()

	// Add CORS middleware
	s.setupCORS()

	// Setup routes
	s.setupRoutes()

	return s, nil
}

func (s *Server) setupCORS() {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   s.config.Server.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	s.router.Use(corsMiddleware.Handler)
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.Get("/health", s.healthHandler)

	// OAuth2 routes
	s.router.Get("/login", loginHandler)
	s.router.Get("/callback", callbackHandler)
	s.router.Get("/logout", logoutHandler)
	s.router.Get("/.well-known/openid-configuration", s.openIDConfiguration)
	s.router.Get("/.well-known/jwks.json", s.jwksHandler)

	// API routes
	s.router.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("hermes_session")
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				data := base64Decode(cookie.Value)
				if data == nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				var session Session
				if err := json.Unmarshal(data, &session); err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// Check if session has expired
				if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// Add auth header to outgoing requests
				req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}

				// Set authorization header
				if session.Token != nil {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", session.Token.AccessToken))
				}

				next.ServeHTTP(w, req)
			})
		})
		
		r.Post("/v1/chat/completions", s.chatCompletionsHandler)
		r.Get("/v1/models", s.modelsHandler)
		r.Get("/v1/models/{model}", s.getModelHandler)
	})

	// Root redirect
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		session := getSession(r)
		if session != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("hermes_session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		data := base64Decode(cookie.Value)
		if data == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if session has expired
		if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add auth header to outgoing requests
		req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set authorization header
		if session.Token != nil {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", session.Token.AccessToken))
		}

		next.ServeHTTP(w, req)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (s *Server) openIDConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issuer":          "http://localhost:8080",
		"authorization_endpoint": "http://localhost:8080/login",
		"token_endpoint": "http://localhost:8080/callback",
		"userinfo_endpoint": "http://localhost:8080/userinfo",
		"jwks_uri":        "http://localhost:8080/.well-known/jwks.json",
		"scopes_supported": []string{"openid", "profile", "email"},
	})
}

func (s *Server) jwksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"alg": "RS256",
				"kty": "RSA",
				"use": "sig",
			},
		},
	})
}

func (s *Server) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request
	var req struct {
		Model      string                 `json:"model"`
		Messages   []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream       bool   `json:"stream"`
		Temperature  float64 `json:"temperature"`
		MaxTokens    int    `json:"max_tokens"`
		TopP         float64 `json:"top_p"`
		ResponseFormat any    `json:"response_format"`
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		http.Error(w, "Messages array cannot be empty", http.StatusBadRequest)
		return
	}

	// Build request for LLM
	llmURL := s.config.Server.LLMAPIURL
	if llmURL == "" {
		llmURL = "http://localhost:11434/api/generate"
	}

	llmReq, err := json.Marshal(map[string]interface{}{
		"model":       req.Model,
		"prompt":      messagesToString(req.Messages),
		"stream":      req.Stream,
		"temperature": req.Temperature,
		"n_predict":   req.MaxTokens,
		"top_p":       req.TopP,
	})
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	// Forward request to LLM
	resp, err := http.Post(llmURL, "application/json", strings.NewReader(string(llmReq)))
	if err != nil {
		http.Error(w, "Failed to forward request to LLM", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response to client
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "LLM API error", resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func (s *Server) modelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":     "local-llm",
				"object": "model",
				"created": 1234567890,
				"owned_by": "local",
			},
		},
	})
}

func (s *Server) getModelHandler(w http.ResponseWriter, r *http.Request) {
	model := chi.URLParam(r, "model")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": model,
		"object": "model",
		"created": 1234567890,
		"owned_by": "local",
	})
}

func messagesToString(messages []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}) string {
	var prompt strings.Builder
	for _, msg := range messages {
		if msg.Role == "user" {
			prompt.WriteString("User: " + msg.Content)
		} else if msg.Role == "assistant" {
			prompt.WriteString("Assistant: " + msg.Content)
		} else {
			prompt.WriteString(msg.Content)
		}
		prompt.WriteString("\n\n")
	}
	return prompt.String()
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(data string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	return decoded
}


// Session holds session data
type Session struct {
	UserInfo map[string]interface{} `json:"user_info,omitempty"`
	Token     *oauth2.Token          `json:"token,omitempty"`
	CreatedAt time.Time              `json:"created_at,omitempty"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
}

func getSession(r *http.Request) *Session {
	cookie, err := r.Cookie("hermes_session")
	if err != nil {
		return nil
	}

	data := base64Decode(cookie.Value)
	if data == nil {
		return nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	// Check if session has expired
	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil
	}

	return &session
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement OAuth2 login
	http.Redirect(w, r, "/login", http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement OAuth2 callback
	http.Redirect(w, r, "/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "hermes_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
