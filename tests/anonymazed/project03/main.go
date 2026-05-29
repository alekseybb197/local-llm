package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/handlers"
	"github.com/oauth2-proxy/local-llm-proxy/middleware"
	"github.com/oauth2-proxy/local-llm-proxy/oauth"
	"github.com/oauth2-proxy/local-llm-proxy/proxy"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file (JSON)")
	flag.Parse()

	// Load configuration
	cfg := config.DefaultConfig()
	if *configPath != "" {
		if err := loadConfigFile(cfg, *configPath); err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	}

	// Initialize database
	log.Println("Initializing database...")
	database, err := db.NewDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize OAuth server
	log.Println("Initializing OAuth server...")
	oauthServer := oauth.NewOAuthServer(database, &cfg.OAuth)

	// Initialize middleware
	log.Println("Initializing middleware...")
	mw := middleware.NewMiddleware(database, cfg)

	// Initialize proxy
	log.Println("Initializing proxy...")
	proxyHandler := proxy.NewProxy(cfg)

	// Create router
	r := chi.NewRouter()

	// Apply middleware
	r.Use(mw.Recovery)
	r.Use(mw.Logging)
	// Skip auth middleware for protected routes - handled per-route

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           int(cfg.CORS.MaxAge.Seconds()),
	}))

	// Health check endpoints (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
			"reason": "Service is ready",
		})
	})

	// OAuth endpoints (no auth)
	r.Get("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		oauthServer.Authorize(w, r)
	})

	r.Get("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		oauthServer.Callback(w, r)
	})

	// Token endpoint
	r.Post("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		handler := handlers.NewTokenHandler(database, oauthServer, cfg)
		handler.Token(w, r)
	})

	// Client management endpoints
	r.Post("/oauth/client/register", func(w http.ResponseWriter, r *http.Request) {
		handler := handlers.NewTokenHandler(database, oauthServer, cfg)
		handler.RegisterClient(w, r)
	})

	r.Delete("/oauth/client/delete", func(w http.ResponseWriter, r *http.Request) {
		handler := handlers.NewTokenHandler(database, oauthServer, cfg)
		handler.DeleteClient(w, r)
	})

	r.Get("/oauth/client/info", func(w http.ResponseWriter, r *http.Request) {
		handler := handlers.NewTokenHandler(database, oauthServer, cfg)
		handler.ClientInfo(w, r)
	})

	// Proxy endpoints - forward all other requests to LLM
	proxyPath := "/api/v1"
	r.Group(func(r chi.Router) {
		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				// Skip auth for proxy endpoints
				req = req.WithContext(context.WithValue(req.Context(), "skipAuth", true))
				h.ServeHTTP(w, req)
			})
		})
		r.Handle(proxyPath+"/chat/completions", proxyHandler)
		r.Handle(proxyPath+"/embeddings", proxyHandler)
		r.Handle(proxyPath+"/completions", proxyHandler)
	})

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "OAuth2 Proxy for LLM",
			"version":     "1.0.0",
			"description": "Production-grade OAuth2 proxy with Authorization Code Flow",
			"endpoints": map[string]string{
				"health":         "/health",
				"ready":          "/ready",
				"oauth_authorize": "/oauth/authorize",
				"oauth_callback": "/oauth/callback",
				"oauth_token":    "/oauth/token",
				"register_client": "/oauth/client/register",
				"delete_client":  "/oauth/client/delete",
				"client_info":    "/oauth/client/info",
				"proxy":          proxyPath + "/chat/completions",
			},
		})
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-shutdown

	log.Println("Shutting down server...")

	// Create grace period for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfigFile(cfg *config.Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return json.Unmarshal(data, cfg)
}
