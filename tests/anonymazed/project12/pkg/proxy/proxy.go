package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout     = 30 * time.Second
	defaultMaxRetries  = 3
	defaultRetryDelay  = 100 * time.Millisecond
)

type LLMProxy struct {
	proxyURL string
	client   *http.Client
}

func New(proxyURL string) (*LLMProxy, error) {
	if proxyURL == "" {
		proxyURL = "http://localhost:11434/v1"
	}

	_, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	client := &http.Client{
		Timeout:   defaultTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &LLMProxy{
		proxyURL: proxyURL,
		client:   client,
	}, nil
}

func (p *LLMProxy) Handle(w http.ResponseWriter, r *http.Request, path string) {
	ctx := r.Context()

	// Get API key from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	// Extract API key (remove "Bearer " prefix if present)
	apiKey := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		apiKey = authHeader[7:]
	}

	// Verify API key (in production, verify against database)
	if apiKey == "" {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Create request with API key
	newReq, err := p.createRequest(r, apiKey)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Track usage
	usage := &Usage{
		APIKeyID: apiKey,
		Method:    r.Method,
		Endpoint:  path,
	}

	// Execute request with retries
	var resp *http.Response
	var errVal error

	for retries := 0; retries < defaultMaxRetries; retries++ {
		resp, errVal = p.client.Do(newReq.WithContext(ctx))
		if errVal == nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
		time.Sleep(defaultRetryDelay * time.Duration(retries+1))
	}

	if errVal != nil {
		log.Printf("Request failed: %v", errVal)
		http.Error(w, "Request failed", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Read and track response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	usage.DurationMS = int(time.Since(time.Now()).Milliseconds())
	usage.RequestSize = 0 // Request body already consumed
	usage.ResponseSize = len(body)

	// Store usage (in production, store in database)
	p.storeUsage(usage)

	// Write response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (p *LLMProxy) verifyAPIKey(key string) (string, error) {
	// In production, verify API key against database
	// For demo, accept any non-empty key
	if key == "" {
		return "", fmt.Errorf("empty API key")
	}
	return "default-key", nil
}

func (p *LLMProxy) createRequest(r *http.Request, apiKey string) (*http.Request, error) {
	// Create new request with API key
	newReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}

	// Add API key
	newReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return newReq, nil
}

func (p *LLMProxy) storeUsage(usage *Usage) {
	// In production, store usage in database
}

// Usage represents API key usage
type Usage struct {
	APIKeyID      string
	Method        string
	Endpoint      string
	DurationMS    int
	RequestSize   int
	ResponseSize  int
}
