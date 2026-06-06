package proxy

import (
	"encoding/json"
	"testing"
)

func TestChatRequest_MarshalJSON(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	data := ChatRequest{
		Model:       "llama2",
		Messages:    msgs,
		Temperature: 0.7,
		MaxTokens:   256,
	}

	body, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}

func TestChatResponse_MarshalJSON(t *testing.T) {
	resp := ChatResponse{
		ID:   "test-id",
		Object: "chat",
		Created: 1234567890,
		Model: "test-model",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}

func TestChoice_MarshalJSON(t *testing.T) {
	choice := Choice{
		Index: 0,
		Message: Message{
			Role:    "assistant",
			Content: "Hello!",
		},
		FinishReason: "stop",
	}

	body, err := json.Marshal(choice)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}

func TestMessage_MarshalJSON(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello!",
	}

	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}

func TestUsage_MarshalJSON(t *testing.T) {
	usage := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	body, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Expected non-empty body")
	}
}
