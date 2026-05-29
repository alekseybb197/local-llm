// Package proxy implements HTTP proxy functionality for OAuth2 authorization.
package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProxy handles OpenAI-compatible API requests.
type OpenAIProxy struct {
	proxy *Proxy
}

// ChatCompletionRequest represents an OpenAI chat completion request.
type ChatCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Tools       []Tool            `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty   float64   `json:"presence_penalty,omitempty"`
	N             int               `json:"n,omitempty"`
	ResponseFormat *ResponseFormat   `json:"response_format,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Tool represents a tool for the model.
type Tool struct {
	Type     string    `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function tool.
type Function struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  Parameters
}

// Parameters represents function parameters.
type Parameters struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a schema property.
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ResponseFormat represents a response format.
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatCompletionResponse represents an OpenAI chat completion response.
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []Choice               `json:"choices"`
	Usage             Usage                  `json:"usage"`
	SystemFingerprint string                 `json:"system_fingerprint"`
	Refusal           string                 `json:"refusal,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProxy creates a new OpenAI proxy.
func NewOpenAIProxy(proxy *Proxy) *OpenAIProxy {
	return &OpenAIProxy{
		proxy: proxy,
	}
}

// ChatCompletionHandler handles /v1/chat/completions requests.
func (p *OpenAIProxy) ChatCompletionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify authorization
	if err := p.verifyAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build upstream URL
	upstreamURL := fmt.Sprintf("%s/v1/chat/completions", p.proxy.config.ProxyConfig.LLMEndpoint)

	// Create upstream request
	upstreamReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
		return
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "application/json")

	// Add API key to upstream request
	if p.proxy.config.ProxyConfig.APIKey != "" {
		upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.proxy.config.ProxyConfig.APIKey))
	}

	// Make upstream request
	client := &http.Client{}
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "Failed to connect to upstream service", http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	// Forward response headers
	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(upstreamResp.StatusCode)

	// Read and forward response body
	responseBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Try to parse the response
	var resp ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &resp); err == nil {
		// Add OpenAI-specific headers
		w.Header().Set("Openai-Processing-Ms", "0")
		w.Header().Set("Openai-Organization", "proxy")
		w.Header().Set("Openai-Version", "2020-10-01")
		w.Header().Set("X-Ratelimit-Limit-Tokens", "0")
		w.Header().Set("X-Ratelimit-Remaining-Tokens", "0")
		w.Header().Set("X-Ratelimit-Reset-Tokens", "0")
	}

	w.Write(responseBody)
}

// CompletionsHandler handles /v1/completions requests.
func (p *OpenAIProxy) CompletionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify authorization
	if err := p.verifyAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build upstream URL
	upstreamURL := fmt.Sprintf("%s/v1/completions", p.proxy.config.ProxyConfig.LLMEndpoint)

	// Create upstream request
	upstreamReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
		return
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "application/json")

	// Add API key to upstream request
	if p.proxy.config.ProxyConfig.APIKey != "" {
		upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.proxy.config.ProxyConfig.APIKey))
	}

	// Make upstream request
	client := &http.Client{}
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "Failed to connect to upstream service", http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	// Forward response headers
	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(upstreamResp.StatusCode)

	// Read and forward response body
	responseBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	w.Write(responseBody)
}

// ModelsHandler handles /v1/models requests.
func (p *OpenAIProxy) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify authorization
	if err := p.verifyAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Build upstream URL
	upstreamURL := fmt.Sprintf("%s/v1/models", p.proxy.config.ProxyConfig.LLMEndpoint)

	// Create upstream request
	upstreamReq, err := http.NewRequest(http.MethodGet, upstreamURL, nil)
	if err != nil {
		http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
		return
	}

	upstreamReq.Header.Set("Accept", "application/json")

	// Add API key to upstream request
	if p.proxy.config.ProxyConfig.APIKey != "" {
		upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.proxy.config.ProxyConfig.APIKey))
	}

	// Make upstream request
	client := &http.Client{}
	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "Failed to connect to upstream service", http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	// Forward response headers
	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(upstreamResp.StatusCode)

	// Read and forward response body
	responseBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	w.Write(responseBody)
}

// verifyAuth verifies the authorization of the request.
func (p *OpenAIProxy) verifyAuth(r *http.Request) error {
	// Check if user is authenticated
	if r.Header.Get("Authorization") == "" {
		return fmt.Errorf("missing Authorization header")
	}

	// Check if session exists
	session, err := p.proxy.store.Get(r.Header.Get("X-User-ID"))
	if err != nil {
		return fmt.Errorf("session not found")
	}

	// Check if session is expired
	if time.Now().After(session.Expires) {
		return fmt.Errorf("session expired")
	}

	return nil
}
