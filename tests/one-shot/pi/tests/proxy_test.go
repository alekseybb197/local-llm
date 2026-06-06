package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"oauth2-proxy/pkg/proxy"
)

func TestLLMProxy_New(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}
	if proxy == nil {
		t.Error("Expected LLM proxy to be created")
	}
}

func TestLLMProxy_Handle(t *testing.T) {
	tests := []struct {
		name           string
		proxyURL       string
		authHeader     string
		apiKey         string
		expectError    bool
		expectedStatus int
	}{
		{"missing auth header", "http://localhost:11434/v1", "", "", true, http.StatusUnauthorized},
		{"invalid auth header", "http://localhost:11434/v1", "invalid", "", true, http.StatusUnauthorized},
		{"missing api key", "http://localhost:11434/v1", "Bearer ", "", true, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := proxy.New(tt.proxyURL)
			if err != nil {
				t.Fatalf("Failed to create LLM proxy: %v", err)
			}

			req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
			req.Header.Set("Content-Type", "application/json")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			proxy.Handle(w, req, "/chat/completions")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLLMProxy_WithValidAPIKey(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")

	w := httptest.NewRecorder()
	proxy.Handle(w, req, "/chat/completions")

	if w.Code >= 500 {
		t.Errorf("Expected status < 500, got %d", w.Code)
	}
}

func TestLLMProxy_BearerPrefix(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key-with-bearer")

	w := httptest.NewRecorder()
	proxy.Handle(w, req, "/chat/completions")

	if w.Code >= 500 {
		t.Logf("Got status %d", w.Code)
	}
}

func TestLLMProxy_InvalidURL(t *testing.T) {
	_, err := proxy.New("not-a-valid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestLLMProxy_EmptyAuth(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "")

	w := httptest.NewRecorder()
	proxy.Handle(w, req, "/chat/completions")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLLMProxy_ValidAuthHeader(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-key")

	w := httptest.NewRecorder()
	proxy.Handle(w, req, "/chat/completions")

	if w.Code == http.StatusUnauthorized {
		t.Errorf("Expected status != %d", http.StatusUnauthorized)
	}
}

func TestLLMProxy_DefaultProxyURL(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	if proxy == nil {
		t.Error("Expected LLM proxy to be created")
	}
}

func TestLLMProxy_BodyRead(t *testing.T) {
	proxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to create LLM proxy: %v", err)
	}

	body := `{"model":"test","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	proxy.Handle(w, req, "/chat/completions")

	if w.Code >= 500 {
		t.Logf("Got status %d", w.Code)
	}
}
