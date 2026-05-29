package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProxy_NewProxy(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	if p.languageModelURL != "http://localhost:11434/v1" {
		t.Errorf("Expected URL 'http://localhost:11434/v1', got '%s'", p.languageModelURL)
	}

	if p.httpClient == nil {
		t.Fatal("Expected httpClient to be set")
	}

	if p.httpClient.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s, got %v", p.httpClient.Timeout)
	}
}

func TestProxy_CreateRequest(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	msgs := []Message{
		{Role: "user", Content: "Hello"},
	}

	req, err := p.CreateRequest("llama2", msgs, 0.7, 256)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	if req.URL.Path != "/v1/chat/completions" {
		t.Errorf("Expected path '/v1/chat/completions', got %s", req.URL.Path)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", req.Header.Get("Content-Type"))
	}
}

func TestProxy_ProxyRequest(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	// Create a mock response
	mockBody := []byte(`{"id":"test","object":"chat","created":123,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)

	// Create a server that returns the mock response
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mockBody)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p.languageModelURL = server.URL

	// Create request
	req, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(mockBody))

	proxyResp, err := p.ProxyRequest(req)
	if err != nil {
		t.Fatalf("Failed to proxy request: %v", err)
	}

	if proxyResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, proxyResp.StatusCode)
	}

	if len(proxyResp.Body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}

func TestProxy_Chat(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	// Create a mock response
	mockResp := ChatResponse{
		ID:   "test-id",
		Object: "chat",
		Created: time.Now().Unix(),
		Model: "test-model",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	body, _ := json.Marshal(mockResp)

	// Create a server that returns the mock response
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p.languageModelURL = server.URL

	// Create request
	req, _ := p.CreateRequest("test-model", []Message{
		{Role: "user", Content: "Hello"},
	}, 0.7, 256)

	chatResp, err := p.Chat(req)
	if err != nil {
		t.Fatalf("Failed to chat: %v", err)
	}

	if chatResp.ID != mockResp.ID {
		t.Errorf("Expected ID '%s', got '%s'", mockResp.ID, chatResp.ID)
	}
}

func TestProxy_Chat_InvalidResponse(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	// Create a server that returns invalid JSON
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p.languageModelURL = server.URL

	// Create request
	req, _ := p.CreateRequest("test-model", []Message{
		{Role: "user", Content: "Hello"},
	}, 0.7, 256)

	_, err := p.Chat(req)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestProxy_ChatHandler(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	// Mock response
	mockResp := ChatResponse{
		ID:   "test-id",
		Object: "chat",
		Created: time.Now().Unix(),
		Model: "test-model",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	body, _ := json.Marshal(mockResp)

	// Create a server that returns the mock response
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p.languageModelURL = server.URL

	// Create request
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"test-model","messages":[{"role":"user","content":"Hello"}],"temperature":0.7,"max_tokens":256}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	p.ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.ID != mockResp.ID {
		t.Errorf("Expected ID '%s', got '%s'", mockResp.ID, resp.ID)
	}
}

func TestProxy_ChatHandler_InvalidMethod(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	req := httptest.NewRequest("GET", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()

	p.ChatHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestProxy_ChatHandler_InvalidBody(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	p.ChatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxy_GetModelList(t *testing.T) {
	p := NewProxy("http://localhost:11434/v1")

	// Mock models response
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		modelsData := []struct {
			ID string `json:"id"`
		}{
			{"llama2"},
			{"mistral"},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data":   modelsData,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p.languageModelURL = server.URL

	models, err := p.GetModelList()
	if err != nil {
		t.Fatalf("Failed to get model list: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0] != "llama2" {
		t.Errorf("Expected first model 'llama2', got '%s'", models[0])
	}
}
