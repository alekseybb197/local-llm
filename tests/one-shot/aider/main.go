package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

var (
	// Config defaults
	DefaultLLMURL      = "http://localhost:11434/v1"
	DefaultClientID    = "local_llm_proxy"
	DefaultClientSecret = "super_secret_key_change_in_prod"
	DefaultRedirectURI = "http://localhost:8080/callback"
	DefaultIssuer      = "http://localhost:8080/auth"
	DefaultAudience    = "local_llm_proxy"
	DefaultExpiry      = 15 * time.Minute
)

func main() {
	// Load config from environment or use defaults
	config := LoadConfig()

	// Generate RSA key pair for signing tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Setup OAuth2 Client
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURI,
		Scopes:       []string{"openid", "profile", "llm_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.Issuer + "/authorize",
			TokenURL: config.Issuer + "/token",
		},
	}

	// Setup Router
	r := mux.NewRouter()

	// OAuth Endpoints
	r.HandleFunc("/authorize", handleAuthorize).Methods("GET")
	r.HandleFunc("/callback", handleCallback).Methods("GET")
	r.HandleFunc("/auth/token", handleTokenExchange).Methods("POST")
	r.HandleFunc("/auth/refresh", handleRefresh).Methods("POST")

	// Health Check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Proxy Endpoint
	r.PathPrefix("/v1/").Handler(http.StripPrefix("/v1/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHandler(w, r, privateKey, config.LLMURL)
	})))

	// Root redirect
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusFound)
	})

	// CORS Middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // In production, restrict this
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Accept"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
	)(r)

	// Start Server
	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Starting OAuth2 Proxy on %s", addr)
	log.Printf("LLM Endpoint: %s", config.LLMURL)
	log.Printf("Client ID: %s", config.ClientID)

	// If TLS cert is provided, use it
	if config.TLSCert != "" && config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
		if err != nil {
			log.Fatalf("Failed to load TLS cert: %v", err)
		}
		log.Printf("Starting HTTPS on %s", addr)
		log.Fatal(http.ListenAndServe(addr, corsHandler))
	} else {
		log.Fatal(http.ListenAndServe(addr, corsHandler))
	}
}

// LoadConfig reads environment variables
func LoadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		LLMURL:      getEnv("LLM_URL", DefaultLLMURL),
		ClientID:    getEnv("CLIENT_ID", DefaultClientID),
		ClientSecret: getEnv("CLIENT_SECRET", DefaultClientSecret),
		RedirectURI: getEnv("REDIRECT_URI", DefaultRedirectURI),
		Issuer:      getEnv("ISSUER", DefaultIssuer),
		Audience:    getEnv("AUDIENCE", DefaultAudience),
		Expiry:      getEnv("TOKEN_EXPIRY", DefaultExpiry),
		TLSCert:     getEnv("TLS_CERT", ""),
		TLSKey:      getEnv("TLS_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

type Config struct {
	Port        string
	LLMURL      string
	ClientID    string
	ClientSecret string
	RedirectURI string
	Issuer      string
	Audience    string
	Expiry      string
	TLSCert     string
	TLSKey      string
}
