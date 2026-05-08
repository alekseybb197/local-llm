package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// llmRequest represents a request to the LLM
type llmRequest struct {
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Context  []int                  `json:"context,omitempty"`
	System   string                 `json:"system,omitempty"`
	Format   string                 `json:"format,omitempty"`
	KeepAlive *time.Duration         `json:"keep_alive,omitempty"`
}

// llmResponse represents the LLM response
type llmResponse struct {
	Model       string      `json:"model"`
	CreatedAt   time.Time   `json:"created_at"`
	Done        bool        `json:"done"`
	TotalDuration time.Duration `json:"total_duration,omitempty"`
	LoadDuration time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount int     `json:"eval_count,omitempty"`
	EvalDuration time.Duration `json:"eval_duration,omitempty"`
	Response    string      `json:"response"`
	Error       string      `json:"error,omitempty"`
}

// llmGenerateResponse for streaming
type llmGenerateResponse struct {
	Model       string      `json:"model"`
	Done        bool        `json:"done"`
	TotalDuration time.Duration `json:"total_duration,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount int     `json:"eval_count,omitempty"`
	EvalDuration time.Duration `json:"eval_duration,omitempty"`
	Response    string      `json:"response"`
	DoneReason  string      `json:"done_reason,omitempty"`
}

// llmClient handles communication with the LLM
type llmClient struct {
	host     string
	port     int
	path     string
	token    string
	baseURL  string
}

func newLLMClient(host, port, token string) *llmClient {
	return &llmClient{
		host: host,
		port: port,
		token: token,
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
	}
}

// generate sends a request to the LLM
func (c *llmClient) generate(req llmRequest) (*llmResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL := c.baseURL + c.path
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var response llmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// stream sends a streaming request to the LLM
func (c *llmClient) stream(req llmRequest) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := c.baseURL + c.path
	reqURL = strings.Replace(reqURL, "/api/generate", "/api/generate?stream=true", 1)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp.Body, nil
}

// chat sends a chat request to the LLM
func (c *llmClient) chat(req map[string]interface{}) (*llmResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL := c.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var response llmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// embed sends an embedding request to the LLM
func (c *llmClient) embed(prompt string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqURL := c.baseURL + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse embeddings (format depends on LLM)
	// This is a simplified example - actual parsing may vary
	return nil, nil
}
