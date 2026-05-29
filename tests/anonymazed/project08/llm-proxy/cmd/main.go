package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-proxy/config"
	"llm-proxy/proxy"
	"llm-proxy/server"
	"llm-proxy/storage"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set defaults from environment
	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("OAUTH_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = os.Getenv("OAUTH_CLIENT_SECRET")
	}
	if cfg.RedirectURL == "" {
		cfg.RedirectURL = os.Getenv("OAUTH_REDIRECT_URL")
	}
	if cfg.AuthorizationURL == "" {
		cfg.AuthorizationURL = os.Getenv("OAUTH_AUTH_URL")
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = os.Getenv("OAUTH_TOKEN_URL")
	}
	if cfg.LLMAPIURL == "" {
		cfg.LLMAPIURL = os.Getenv("LLM_API_URL")
	}

	// Initialize storage
	store := storage.NewMemoryStore()

	// Initialize server
	srv := server.NewServer(cfg, store)

	// Start server in goroutine
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Server started on %s", cfg.ListenAddr)
	log.Printf("LLM API URL: %s", cfg.LLMAPIURL)
	log.Printf("OAuth Authorization URL: %s", cfg.AuthorizationURL)
	log.Printf("OAuth Token URL: %s", cfg.TokenURL)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	srv.Cleanup()
}

// Helper function for proxy
func ProxyRequest(targetURL string, originalRequest *http.Request, headers string, timeout time.Duration) (*proxy.ProxyResponse, error) {
	return proxy.ProxyRequest(targetURL, originalRequest, headers, timeout)
}
