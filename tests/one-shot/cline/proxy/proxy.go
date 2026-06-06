package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/models"
)

type Proxy struct {
	config     *config.Config
	httpClient *http.Client
}

func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.LLM.Timeout,
		},
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get token from context
	token, ok := r.Context().Value("token").(*models.Token)
	if !ok {
		http.Error(w, "Missing token in context", http.StatusUnauthorized)
		return
	}

	// Get LLM URL from context or use default
	llmURL := r.URL.Query().Get("llm_url")
	if llmURL == "" {
		llmURL = p.config.LLM.LLMURL
	}

	// Check rate limit
	clientIP := getClientIP(r)
	if !p.checkRateLimit(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Forward request to LLM
	proxyURL := fmt.Sprintf("%s%s", llmURL, r.URL.Path)

	log.Printf("Forwarding %s %s -> %s", r.Method, r.URL.Path, proxyURL)

	// Forward the request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Set content length
	req.ContentLength = r.ContentLength

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}

func (p *Proxy) checkRateLimit(clientIP string) bool {
	// Simple rate limiting implementation
	// In production, use a proper rate limiting library
	return true
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	return r.RemoteAddr
}

func (p *Proxy) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (p *Proxy) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	// Check if LLM is reachable
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/health", p.config.LLM.LLMURL))
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"reason": "LLM unreachable",
		})
		return
	}
	resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
		"reason": "LLM is reachable",
	})
}