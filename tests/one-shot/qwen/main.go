package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config := DefaultConfig()
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	stateMgr := NewStateManager(config.StateSecret, config.SessionTimeout)
	sessionStore := NewSessionStore()
	llmClient := NewLLMClient(config.LLMAPIURL, config.LLMAPIKey)
	handler := NewHandler(config, stateMgr, llmClient, sessionStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.Home)
	mux.HandleFunc("/auth", handler.Auth)
	mux.HandleFunc("/callback", handler.Callback)
	mux.HandleFunc("/v1/", handler.Proxy)
	mux.HandleFunc("/health", handler.Health)

	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 OAuth2 Proxy started on :%s", config.ServerPort)
		log.Printf("📝 API endpoint: %s", config.LLMAPIURL)
		log.Printf("🔗 Auth: /auth, Callback: /callback")
		log.Printf("💚 Health: /health")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
