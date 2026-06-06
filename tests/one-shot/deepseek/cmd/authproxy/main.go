package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deepseek/oauth2-proxy/internal/pkg"
)

func main() {
	// Load configuration
	config := pkg.LoadConfig()

	// Initialize state machine
	stateMachine := pkg.NewStateMachine()

	// Initialize handler
	handler := pkg.NewHandler(stateMachine, *config)

	// Create HTTP server
	server := &http.Server{
		Addr:         config.ServerPort,
		Handler:      createServerHandler(handler),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create TLS configuration
	tlsConfig := createTLSConfig()

	// Start server in goroutine
	go func() {
		log.Printf("Starting OAuth2 Proxy server on %s", config.ServerPort)

		// Start HTTPS server
		httpsServer := &http.Server{
			Addr:      config.ServerPort,
			TLSConfig: tlsConfig,
			Handler:   createServerHandler(handler),
		}

		if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("TLS server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

// createServerHandler creates the main HTTP handler
func createServerHandler(handler *pkg.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth2/auth", func(w http.ResponseWriter, r *http.Request) {
		handler.StartAuthorization(w, r)
	})

	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		handler.Callback(w, r)
	})

	mux.HandleFunc("/oauth2/token", handler.TokenEndpoint)
	mux.HandleFunc("/oauth2/refresh", handler.RefreshEndpoint)
	mux.HandleFunc("/oauth2/.well-known/oauth-authorization-server", handler.InfoEndpoint)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OAuth2 Proxy"))
	})

	return mux
}

// createTLSConfig creates TLS configuration for HTTPS
func createTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		NextProtos: []string{"h2", "http/1.1"},
	}
}

// generateSelfSignedCert generates a self-signed certificate for testing
func generateSelfSignedCert() (*rsa.PrivateKey, *x509.Certificate) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Generate certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"OAuth2 Proxy"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	return privateKey, cert
}

// loadCertFromPEM loads a certificate from PEM file
func loadCertFromPEM(certFile, keyFile string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return cert, nil
}
