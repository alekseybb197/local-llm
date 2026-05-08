package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultLLMURL     = "http://localhost:11434"
	defaultOAuthURL   = "http://localhost:9000/oauth"
	defaultClientID   = "llm-proxy-client"
	defaultClientSecret = "client-secret-key"
)

var (
	allowedIPs = map[string]bool{
		"127.0.0.1": true,
		"::1":       true,
	}
)

func main() {
	config := &oauth2.Config{
		ClientID:     defaultClientID,
		ClientSecret: defaultClientSecret,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	// Get OAuth token
	token, err := getOAuthToken(config)
	if err != nil {
		log.Fatalf("Failed to get OAuth token: %v", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url.URL{Scheme: "http", Host: defaultLLMURL})

	// Customize proxy director to add auth header
	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}

	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // For local LLM development
		},
	}

	// Create handler chain
	handler := createAuthHandler(config, token, proxy)

	fmt.Println("OAuth2 Proxy for Local LLM starting on :8080")
	fmt.Printf("LLM URL: %s\n", defaultLLMURL)
	fmt.Printf("OAuth URL: %s\n", defaultOAuthURL)

	log.Fatal(http.ListenAndServe(":8080", handler))
}

func getOAuthToken(config *oauth2.Config) (*oauth2.Token, error) {
	// For local development, create a client credentials flow
	// In production, you would have a proper OAuth server
	clientCredsConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     "http://localhost:9000/oauth/token",
		Scopes:       config.Scopes,
	}

	token, err := clientCredsConfig.Token()
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		RefreshToken: token.RefreshToken,
	}, nil
}

func createAuthHandler(config *oauth2.Config, token *oauth2.Token, proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request has auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Verify token
		tokenPart := strings.TrimPrefix(authHeader, "Bearer ")
		if !validateToken(tokenPart) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Create context with OAuth token
		ctx := r.Context()
		// Add OAuth token to context for downstream use
		ctx = oauth2.SetAuth(ctx, oauth2.AccessToken(tokenPart))

		// Create request copy
		newReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Add headers
		for k, vv := range r.Header {
			newReq.Header[k] = vv
		}

		// Set auth header for LLM
		newReq.Header.Set("Authorization", authHeader)
		newReq.Header.Set("X-OAuth-Token", tokenPart)

		// Execute proxy
		proxy.ServeHTTP(w, newReq)
	})
}

func validateToken(token string) bool {
	// For production, verify token signature and claims
	// For local development, simple check
	return len(token) > 0 && strings.HasPrefix(token, "Bearer ") || len(token) > 50
}

func generateCert() (string, error) {
	// Generate self-signed cert for local dev
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	template := x509.Certificate{
		SerialNumber: uint64(time.Now().Unix()),
		Subject: x509.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", err
	}

	var certPEM, keyPEM []byte
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	certFile := "/tmp/llm-proxy.crt"
	keyFile := "/tmp/llm-proxy.key"

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return "", err
	}

	return certFile, nil
}
