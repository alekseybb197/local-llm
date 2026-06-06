package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test_verifier_123"
	challenge := generateCodeChallenge(verifier)
	
	// Verify it's not empty
	if challenge == "" {
		t.Error("Challenge should not be empty")
	}
}

func TestValidateToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_123",
		"aud": DefaultAudience,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Validate
	_, err = validateToken(tokenString, privateKey)
	if err != nil {
		t.Errorf("Valid token should not fail validation: %v", err)
	}

	// Test expired token
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_123",
		"aud": DefaultAudience,
		"exp": time.Now().Add(-15 * time.Minute).Unix(),
	})

	expiredString, _ := expiredToken.SignedString(privateKey)
	_, err = validateToken(expiredString, privateKey)
	if err == nil {
		t.Error("Expired token should fail validation")
	}
}

func TestProxyHandler(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	
	// Create a mock LLM server
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"delta":{"content":"Hello"}}]}`))
	}))
	defer llmServer.Close()

	config := Config{
		LLMURL: llmServer.URL,
	}

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_123",
		"aud": DefaultAudience,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	tokenString, _ := token.SignedString(privateKey)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	proxyHandler(rr, req, privateKey, config.LLMURL)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestProxyHandlerUnauthorized(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	config := Config{LLMURL: "http://localhost:11434/v1"}

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	rr := httptest.NewRecorder()
	proxyHandler(rr, req, privateKey, config.LLMURL)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	
	// This function in the main code is mocked to generate a JWT.
	// We test that it returns a non-nil token.
	token, err := exchangeCodeForToken("dummy_code")
	if err != nil {
		t.Fatalf("Failed to exchange code: %v", err)
	}
	
	if token.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
}

func TestLoadConfig(t *testing.T) {
	// Test default values
	config := LoadConfig()
	if config.Port != "8080" {
		t.Errorf("Default port should be 8080, got %s", config.Port)
	}
	if config.LLMURL != "http://localhost:11434/v1" {
		t.Errorf("Default LLM URL incorrect")
	}
}

func TestOAuthFlowIntegration(t *testing.T) {
	// Simulate the flow:
	// 1. Generate verifier
	verifier, _ := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)
	
	// 2. Exchange (mocked)
	token, _ := exchangeCodeForToken("valid_code")
	
	// 3. Validate
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	_, err := validateToken(token.AccessToken, privateKey)
	if err != nil {
		t.Errorf("Token validation failed: %v", err)
	}
}
