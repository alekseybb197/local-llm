package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	hermesconfig "hermes/config"
	"hermes/internal/server"
	"hermes/internal/store"
)

func main() {
	// Load configuration
	cfg, err := hermesconfig.LoadFromFile("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set default values
	if cfg.OAuth2.ClientID == "" {
		cfg.OAuth2.ClientID = os.Getenv("HERMES_OAUTH2_CLIENT_ID")
	}
	if cfg.OAuth2.ClientSecret == "" {
		cfg.OAuth2.ClientSecret = os.Getenv("HERMES_OAUTH2_CLIENT_SECRET")
	}
	if cfg.OAuth2.RedirectURI == "" {
		cfg.OAuth2.RedirectURI = fmt.Sprintf("http://%s:%d/callback", cfg.Server.Host, cfg.Server.Port)
	}
	if cfg.Server.SessionSecret == "" {
		cfg.Server.SessionSecret = os.Getenv("HERMES_SERVER_SESSION_SECRET")
	}
	if cfg.Server.CookieExpiration == 0 {
		cfg.Server.CookieExpiration = 1 * time.Hour
	}
	if cfg.Server.AllowedOrigins == nil {
		cfg.Server.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	}

	// Create store
	storeInstance := store.NewInMemoryStore(15 * time.Minute)

	// Create server
	s, err := server.NewServer(cfg, storeInstance)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	serverInstance := &http.Server{
		Addr:         addr,
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", addr)
	log.Printf("OAuth2 login available at http://%s/login", addr)

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := serverInstance.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown: %v", err)
	}

	log.Println("Server stopped")
}
