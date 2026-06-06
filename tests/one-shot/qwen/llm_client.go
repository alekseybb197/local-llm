package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

type LLMClient struct {
	url      string
	apiKey   string
	httpClient *http.Client
	mu       sync.RWMutex
}

func NewLLMClient(url, apiKey string) *LLMClient {
	return &LLMClient{
		url:      url,
		apiKey:   apiKey,
		httpClient: &http.Client{},
	}
}

func (c *LLMClient) ChatCompletions(messages []Message, model string, temperature float64, maxTokens int) (*LLMResponse, error) {
	body, err := json.Marshal(LLMRequest{
		Model:        model,
		Messages:     messages,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return nil, err
	}
	
	reqURL := c.url + "/chat/completions"
	httpReq, err := http.NewRequest("POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &httpError{
			StatusCode: resp.StatusCode,
			Body: string(bodyBytes),
		}
	}
	
	return parseLLMResponse(resp.Body)
}

func parseLLMResponse(body io.Reader) (*LLMResponse, error) {
	var resp LLMResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type httpError struct {
	StatusCode int
	Body       string
	mu         sync.Mutex
}

func (e *httpError) Error() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Body
}
