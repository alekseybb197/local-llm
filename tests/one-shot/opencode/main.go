package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oauth2proxy/handlers"
	"oauth2proxy/middleware"
	"oauth2proxy/proxy"
	"oauth2proxy/store"
)

func main() {
	// Get configuration from environment
	langModelURL := os.Getenv("LLM_URL")
	if langModelURL == "" {
		langModelURL = "http://localhost:11434/v1" // Default to ollama
	}

	// Create stores
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	// Create middleware
	authMW := middleware.NewAuthMiddleware(stateStore, userStore)

	// Create proxy
	llmProxy := proxy.NewProxy(langModelURL)

	// Create HTTP server
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/login", handlers.LoginHandler(stateStore, userStore))
	mux.HandleFunc("/callback", handlers.CallbackHandler(stateStore, userStore))
	mux.HandleFunc("/logout", handlers.LogoutHandler(stateStore))
	mux.HandleFunc("/", handlers.DashboardHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/v1/chat/completions", llmProxy.ChatHandler)
	mux.HandleFunc("/api/v1/models", llmProxy.ModelsHandler)

	// Create server with middleware
	server := &http.Server{
		Addr:         ":8080",
		Handler:      authMW.RequireAuth(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting OAuth2 Proxy on http://localhost:8080")
		log.Printf("LLM URL: %s", langModelURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Create cancellation context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}
