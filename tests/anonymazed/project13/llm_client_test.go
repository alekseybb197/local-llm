package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMClient_ChatCompletions_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
			http.Error(w, "Wrong path", http.StatusBadRequest)
			return
		}
		
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Bearer test-api-key, got %s", auth)
			http.Error(w, "Invalid auth", http.StatusUnauthorized)
			return
		}
		
		body, _ := io.ReadAll(r.Body)
		var req LLMRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal body: %v", err)
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		
		if len(req.Messages) == 0 {
			t.Error("Messages should not be empty")
			http.Error(w, "Messages required", http.StatusBadRequest)
			return
		}
		
		response := LLMResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "test-model",
			Choices: []Choice{
				{
					Index:     0,
					Message:   Message{Role: "assistant", Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	client := NewLLMClient("http://test", "test-api-key")
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", handler)
	
	ts := httptest.NewServer(mux)
	defer ts.Close()
	
	client.url = ts.URL + "/v1"
	
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	
	resp, err := client.ChatCompletions(messages, "test-model", 0.7, 100)
	if err != nil {
		t.Fatalf("ChatCompletions failed: %v", err)
	}
	
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("Expected 'Hello!', got %s", resp.Choices[0].Message.Content)
	}
}

func TestLLMClient_ChatCompletions_BadRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
	})
	
	client := NewLLMClient("http://test", "test-api-key")
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", handler)
	
	ts := httptest.NewServer(mux)
	defer ts.Close()
	
	client.url = ts.URL + "/v1"
	
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	
	_, err := client.ChatCompletions(messages, "test-model", 0.7, 100)
	if err == nil {
		t.Error("Expected error for bad request")
	}
}

func TestLLMClient_ChatCompletions_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	})
	
	client := NewLLMClient("http://test", "wrong-key")
	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", handler)
	
	ts := httptest.NewServer(mux)
	defer ts.Close()
	
	client.url = ts.URL + "/v1"
	
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	
	_, err := client.ChatCompletions(messages, "test-model", 0.7, 100)
	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}
