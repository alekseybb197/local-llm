package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type OAuthState struct {
	State      string
	AuthCode   string
	AccessToken string
	ExpiresAt  time.Time
}

var stateStore = make(map[string]*OAuthState)

const (
	AuthURL     = "http://localhost:3001/oauth2/authorize"
	TokenURL    = "http://localhost:3001/oauth2/token"
	APIURL      = "http://localhost:11434/v1"
	AuthTimeout = 30 * time.Second
)

func main() {
	http.HandleFunc("/authorize", authorizeHandler)
	http.HandleFunc("/token", tokenHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/proxy", proxyHandler)
	
	log.Println("OAuth2 Proxy for Local LLM started on :3000")
	log.Printf("Auth URL: %s", AuthURL)
	log.Printf("API URL: %s", APIURL)
	
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	stateStore[state] = &OAuthState{
		State: state,
	}
	
	cookie := http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute),
		Secure:   false,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	
	redirectURI := r.FormValue("redirect_uri")
	if redirectURI == "" {
		redirectURI = "http://localhost:3000/callback"
	}
	
	http.Redirect(w, r, fmt.Sprintf("%s?redirect_uri=%s", AuthURL, redirectURI), http.StatusSeeOther)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	state := r.FormValue("state")
	
	// Validate state
	if stateStore[state] == nil {
		http.Error(w, "Invalid state", http.StatusForbidden)
		return
	}
	
	delete(stateStore, state)
	
	// Exchange code for token
	resp, err := http.PostForm(TokenURL, map[string][]string{
		"client_id":     {"llm-proxy-client"},
		"client_secret": {"secret-key-123456"},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {r.FormValue("redirect_uri")},
	})
	if err != nil {
		http.Error(w, "Failed to get token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}
	
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		http.Error(w, "Invalid token response", http.StatusInternalServerError)
		return
	}
	
	stateStore[state] = &OAuthState{
		State:      state,
		AuthCode:   code,
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:  time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	
	http.Redirect(w, r, "http://localhost:3000", http.StatusSeeOther)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		GrantType string `json:"grant_type"`
		Code      string `json:"code"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	state, exists := stateStore[req.Code]
	if !exists || time.Now().After(state.ExpiresAt) {
		http.Error(w, "Invalid or expired code", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": state.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(state.ExpiresAt).Seconds()),
	})
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Check authorization
	accessToken := r.Header.Get("Authorization")
	if !strings.HasPrefix(accessToken, "Bearer ") {
		http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}
	
	token := strings.TrimPrefix(accessToken, "Bearer ")
	
	// Try to find valid token
	var state *OAuthState
	for _, s := range stateStore {
		if s.AccessToken == token && !time.Now().After(s.ExpiresAt) {
			state = s
			break
		}
	}
	
	if state == nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}
	
	// Forward request to LLM
	newReq := *r
	newReq.URL.Scheme = "http"
	newReq.URL.Host = "localhost:11434"
	
	resp, err := http.DefaultClient.Do(&newReq)
	if err != nil {
		http.Error(w, "Failed to connect to LLM: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(resp.StatusCode)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(body)
}

func generateState() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
