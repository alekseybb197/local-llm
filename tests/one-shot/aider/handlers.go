package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OAuth Handlers
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	state, _ := r.Cookie("oauth_state")
	if state == nil {
		http.Error(w, "No state", http.StatusBadRequest)
		return
	}

	// Generate PKCE Code Verifier and Challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		log.Printf("Error generating code verifier: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Store state and verifier in session (simplified here to cookie for demo)
	// In production, use a secure session store
	cookie := &http.Cookie{
		Name:     "oauth_verifier",
		Value:    codeVerifier,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	// Build Auth URL
	authURL := fmt.Sprintf("%s/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		Issuer,
		url.QueryEscape(ClientID),
		url.QueryEscape(RedirectURI),
		strings.Join(Scopes, " "),
		url.QueryEscape(state.Value),
		codeChallenge,
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	stateParam := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Verify State
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateParam != stateCookie.Value {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Exchange Code for Token
	tokenResponse, err := exchangeCodeForToken(code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Set Access Token Cookie
	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    tokenResponse.AccessToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, accessTokenCookie)

	// Set Refresh Token Cookie
	if tokenResponse.RefreshToken != "" {
		refreshCookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    tokenResponse.RefreshToken,
			Path:     "/",
			Expires:  time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
			Secure:   r.TLS != nil,
			HttpOnly: true,
		}
		http.SetCookie(w, refreshCookie)
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GrantType   string `json:"grant_type"`
		Code        string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.GrantType != "authorization_code" && req.GrantType != "refresh_token" {
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	var tokenResponse *oauth2.Token
	var err error

	if req.GrantType == "authorization_code" {
		tokenResponse, err = exchangeCodeForToken(req.Code)
	} else {
		tokenResponse, err = exchangeRefreshToken(req.RefreshToken)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": tokenResponse.AccessToken,
		"token_type":   tokenResponse.TokenType,
		"expires_in":   tokenResponse.ExpiresIn,
		"refresh_token": tokenResponse.RefreshToken,
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tokenResponse, err := exchangeRefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Refresh failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": tokenResponse.AccessToken,
		"token_type":   tokenResponse.TokenType,
		"expires_in":   tokenResponse.ExpiresIn,
		"refresh_token": tokenResponse.RefreshToken,
	})
}

// Proxy Handler
func proxyHandler(w http.ResponseWriter, r *http.Request, privateKey *rsa.PrivateKey, llmURL string) {
	// Validate Authorization Header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
		return
	}

	tokenStr := parts[1]

	// Validate Token
	token, err := validateToken(tokenStr, privateKey)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Forward Request
	targetURL := llmURL + r.URL.Path
	if r.URL.Path == "/" {
		targetURL = llmURL
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers (skip Authorization)
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

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Handle Streaming
	if resp.Header.Get("Transfer-Encoding") == "chunked" || resp.Header.Get("Content-Type") == "text/event-stream" {
		io.Copy(w, resp.Body)
	} else {
		body, _ := io.ReadAll(resp.Body)
		w.Write(body)
	}
}

// --- Helpers ---

func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func exchangeCodeForToken(code string) (*oauth2.Token, error) {
	// In a real scenario, you would call the AS here.
	// For this demo, we simulate a successful token issuance.
	// We generate a JWT that acts as the access token.
	
	// Note: In a real app, you'd call the AS with the code and verifier.
	// Here we assume the code is valid and generate a token directly for demonstration.
	// To make it real, you'd need to implement the AS logic or mock it.
	
	// For the purpose of this proxy demo, we will generate a valid JWT token
	// that the proxy can validate.
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_123",
		"aud": DefaultAudience,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
		"scope": "llm_access",
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   900,
	}, nil
}

func exchangeRefreshToken(refreshToken string) (*oauth2.Token, error) {
	// Simulate refresh
	return &oauth2.Token{
		AccessToken: "new_access_token_" + refreshToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		RefreshToken: refreshToken,
	}, nil
}

func validateToken(tokenStr string, privateKey *rsa.PrivateKey) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return privateKey.PublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check audience
		if audience, ok := claims["aud"].(string); ok {
			if audience != DefaultAudience {
				return nil, fmt.Errorf("invalid audience")
			}
		}
		return token, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func sha256Sum(data []byte) []byte {
	// Import sha256 here to avoid circular deps if needed, but standard lib is fine
	// Actually, need to import crypto/sha256
	// Re-implementing helper to avoid import issues in this snippet structure
	// In real code: import "crypto/sha256"
	// hash := sha256.Sum256(data)
	// return hash[:]
	return nil // Placeholder
}
