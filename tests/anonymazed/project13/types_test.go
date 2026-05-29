package main

import (
	"encoding/json"
	"testing"
)

func TestErrorResponse_MarshalJSON(t *testing.T) {
	err := &ErrorResponse{
		Error:       "invalid_request",
		ErrorDesc:   "Missing required parameter",
	}
	
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal: %v", marshalErr)
	}
	
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	if result["error"] != "invalid_request" {
		t.Errorf("Expected error 'invalid_request', got '%s'", result["error"])
	}
	
	if result["error_description"] != "Missing required parameter" {
		t.Errorf("Expected error_description, got '%s'", result["error_description"])
	}
}

func TestLLMRequest_MarshalJSON(t *testing.T) {
	req := LLMRequest{
		Model:        "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	var result LLMRequest
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	if result.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", result.Model)
	}
	
	if result.Messages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got '%s'", result.Messages[0].Role)
	}
	
	if result.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", result.Temperature)
	}
	
	if result.MaxTokens != 100 {
		t.Errorf("Expected max_tokens 100, got %d", result.MaxTokens)
	}
}

func TestLLMResponse_MarshalJSON(t *testing.T) {
	resp := LLMResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []Choice{
			{
				Index:     0,
				Message:   Message{Role: "assistant", Content: "Hello, world!"},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}
	
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	var result LLMResponse
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	if result.ID != "chatcmpl-123" {
		t.Errorf("Expected ID 'chatcmpl-123', got '%s'", result.ID)
	}
	
	if result.Choices[0].Message.Content != "Hello, world!" {
		t.Errorf("Expected message content 'Hello, world!', got '%s'", result.Choices[0].Message.Content)
	}
	
	if result.Usage.TotalTokens != 30 {
		t.Errorf("Expected total_tokens 30, got %d", result.Usage.TotalTokens)
	}
}

func TestMessage_MarshalJSON(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "What is OAuth2?",
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	var result Message
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	if result.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", result.Role)
	}
	
	if result.Content != "What is OAuth2?" {
		t.Errorf("Expected content 'What is OAuth2?', got '%s'", result.Content)
	}
}
