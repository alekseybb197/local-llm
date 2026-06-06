// OAuth2 Proxy Server for Local LLM with OpenAI-compatible endpoint
// Implements Authorization Code Flow with PKCE support

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oauth2-proxy/cmd/proxy-server/db"
	"oauth2-proxy/cmd/proxy-server/handlers"
	"oauth2-proxy/pkg/oauth2"
	"oauth2-proxy/pkg/proxy"
)

var (
	configFile        = flag.String("config", "", "Path to configuration file (YAML)")
	httpAddr          = flag.String("addr", ":8080", "HTTP server address")
	llmProxyURL       = flag.String("llm-proxy", "http://localhost:11434/v1", "LLM proxy URL (OpenAI-compatible)")
	oauthURL          = flag.String("oauth-url", "http://localhost:8081", "OAuth2 authorization URL")
	oauthCallbackURL  = flag.String("oauth-callback", "http://localhost:8080/callback", "OAuth2 callback URL")
	oauthClientID     = flag.String("oauth-client-id", "proxy-client", "OAuth2 client ID")
	oauthClientSecret = flag.String("oauth-client-secret", "proxy-secret", "OAuth2 client secret")
	dbPath            = flag.String("db", "proxy.db", "SQLite database path")
	corsAllowOrigins  = flag.String("cors-origins", "*", "Comma-separated CORS origins")
)

func main() {
	flag.Parse()

	cfg := db.NewConfig(
		*configFile,
		*httpAddr,
		*llmProxyURL,
		*oauthURL,
		*oauthCallbackURL,
		*oauthClientID,
		*oauthClientSecret,
		*dbPath,
		*corsAllowOrigins,
	)

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize database
	storeDB, err := db.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer storeDB.Close()

	// Initialize OAuth2 store
	oauthStore := storeDB.NewOAuth2Store()

	// Initialize OAuth2 server
	oauthServer, err := oauth.NewServer(
		oauthStore,
		*oauthURL,
		*oauthCallbackURL,
		*oauthClientID,
		*oauthClientSecret,
	)
	if err != nil {
		log.Fatalf("Failed to initialize OAuth2 server: %v", err)
	}

	// Initialize LLM proxy
	llmProxy, err := proxy.New(cfg.LLMProxyURL)
	if err != nil {
		log.Fatalf("Failed to initialize LLM proxy: %v", err)
	}

	// Create HTTP mux
	router := handlers.NewRouter(oauthServer, llmProxy, cfg.CORSOrigins)

	// Set up HTTP server
	server := &http.Server{
		Addr:         *httpAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("OAuth2 Proxy Server starting on %s", *httpAddr)
	log.Printf("LLM Proxy URL: %s", *llmProxyURL)
	log.Printf("OAuth2 URL: %s", *oauthURL)
	log.Printf("OAuth2 Callback URL: %s", *oauthCallbackURL)
	
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
