package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"oauth2proxy/models"
)

type Proxy struct {
	languageModelURL string
	httpClient       *http.Client
}

func NewProxy(url string) *Proxy {
	return &Proxy{
		languageModelURL: url,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ProxyChatRequest wraps the request with headers
type ProxyChatRequest struct {
	Headers map[string]string `json:"-"`
	Body    io.Reader         `json:"-"`
}

// ProxyResponse represents the proxied response
type ProxyResponse struct {
	StatusCode int
	Headers     map[string][]string
	Body        []byte
}

// ProxyRequest handles proxying requests to the language model
func (p *Proxy) ProxyRequest(req *http.Request) (*ProxyResponse, error) {
	// Copy headers from the original request
	req.Header.Set("Content-Type", "application/json")

	// Make the request to the language model
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to proxy request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}

// Chat proxies a chat request to the language model
func (p *Proxy) Chat(req *http.Request) (*ChatResponse, error) {
	proxyResp, err := p.ProxyRequest(req)
	if err != nil {
		return nil, err
	}

	if proxyResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy returned non-200 status: %d", proxyResp.StatusCode)
	}

	var response ChatResponse
	if err := json.Unmarshal(proxyResp.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateRequest creates a chat request with proper headers
func (p *Proxy) CreateRequest(model string, messages []Message, temperature float64, maxTokens int) (*http.Request, error) {
	data := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.languageModelURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+models.ClientSecret)

	return req, nil
}

// GetModelList retrieves the list of available models
func (p *Proxy) GetModelList() ([]string, error) {
	req, err := http.NewRequest("GET", p.languageModelURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: %d", resp.StatusCode)
	}

	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	result := make([]string, len(models.Data))
	for i, m := range models.Data {
		result[i] = m.ID
	}

	return result, nil
}

// ChatHandler handles chat completions requests
func (p *Proxy) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create proxy request
	proxyReq, err := p.CreateRequest(chatReq.Model, chatReq.Messages, chatReq.Temperature, chatReq.MaxTokens)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Make proxy request
	proxyResp, err := p.ProxyRequest(proxyReq)
	if err != nil {
		log.Printf("Proxy request failed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send response back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(proxyResp.StatusCode)
	w.Write(proxyResp.Body)
}

// ModelsHandler handles models listing requests
func (p *Proxy) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelsList, err := p.GetModelList()
	if err != nil {
		log.Printf("Failed to get models: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Format response
	response := struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
			Created int64  `json:"created"`
		} `json:"data"`
	}{
		Object: "list",
		Data: make([]struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
			Created int64  `json:"created"`
		}, len(modelsList)),
	}

	for i, m := range modelsList {
		response.Data[i] = struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
			Created int64  `json:"created"`
		}{
			ID:      m,
			OwnedBy: "proxy",
			Created: time.Now().Unix(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ChatRequest represents the OpenAI-compatible chat request
type ChatRequest struct {
	Model       string   `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64  `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents the OpenAI-compatible chat response
type ChatResponse struct {
	ID                string    `json:"id"`
	Object            string    `json:"object"`
	Created           int64     `json:"created"`
	Model             string    `json:"model"`
	Choices           []Choice  `json:"choices"`
	Usage             Usage     `json:"usage"`
	SystemFingerprint string    `json:"system_fingerprint,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
