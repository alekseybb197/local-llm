package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	Issuer      = DefaultIssuer
	ClientID    = DefaultClientID
	RedirectURI = DefaultRedirectURI
	Scopes      = []string{"openid", "profile", "llm_access"}
)

// Helper to generate SHA256 hash for PKCE
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// Helper to read body for proxying
func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// Helper to reset body after reading
func resetBody(r *http.Request, body []byte) {
	r.Body = io.NopCloser(strings.NewReader(string(body)))
}

// Helper to handle streaming responses
func streamResponse(w http.ResponseWriter, r *http.Request, client *http.Client, targetURL string) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, vv := range r.Header {
		if k == "Authorization" {
			continue
		}
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}
