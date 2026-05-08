package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config holds OAuth2 proxy configuration
type Config struct {
	HTTPAddr          string
	TrustedCAs        []byte
	JWKSURL           string
	RequiredClaims    []string
	AllowedAudiences  []string
	AllowedHosts      []string
	AllowUnsigned     bool
	AllowedMethods    []string
	AllowedPathPrefix string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: oauth2-proxy <config.json>")
		os.Exit(1)
	}

	config, err := loadConfig(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting OAuth2 Proxy on %s", config.HTTPAddr)
	log.Fatal(http.ListenAndServe(config.HTTPAddr, newProxy(config)))
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.HTTPAddr == "" {
		config.HTTPAddr = ":8080"
	}
	if config.JWKSURL == "" {
		config.JWKSURL = "http://localhost:9000/.well-known/jwks.json"
	}
	if config.AllowedMethods == nil {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	}
	if config.AllowedPathPrefix == "" {
		config.AllowedPathPrefix = "/"
	}

	return &config, nil
}

func newProxy(config *Config) http.Handler {
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/oauth2/callback", handleCallback(config))
	mux.HandleFunc("/oauth2/token", handleToken(config))

	// Protected proxy
	mux.HandleFunc(config.AllowedPathPrefix, handleProxy(config))

	return mux
}

func handleCallback(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code", http.StatusUnauthorized)
			return
		}

		// Exchange code for token
		// In real implementation, exchange code for access token
		// This is a simplified example

		http.Redirect(w, r, "http://localhost:11434/v1/chat/completions", http.StatusFound)
	}
}

func handleToken(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In production, use JWKS to verify tokens
		// This is a simplified example

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": "test_token", "token_type": "Bearer", "expires_in": 3600}`))
	}
}

func handleProxy(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		if !config.ValidateRequest(r) {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		// Get authorization from request
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")

		// Verify token signature (in production, validate against JWKS)
		if !config.ValidateToken(token) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Forward request to LLM
		req, err := http.NewRequest(r.Method, "http://localhost:11434/v1/chat/completions", r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		req.Header = make(http.Header)
		for k, vv := range r.Header {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to proxy request", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func generateClientCert() (*rsa.PrivateKey, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Generate certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"OAuth2 Proxy"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Write to disk
	os.WriteFile("client.crt", certPEM, 0644)
	os.WriteFile("client.key", keyPEM, 0600)

	return privateKey, nil
}

func (c *Config) ValidateRequest(r *http.Request) bool {
	// Check method
	method := strings.ToUpper(r.Method)
	if !contains(c.AllowedMethods, method) {
		return false
	}

	// Check host
	host := r.Host
	if !contains(c.AllowedHosts, host) && len(c.AllowedHosts) > 0 {
		return false
	}

	return true
}

func (c *Config) ValidateToken(token string) bool {
	// In production, verify token signature using JWKS
	// This is a simplified example

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
