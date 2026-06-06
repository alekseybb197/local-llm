package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	config      *Config
	stateMgr    *StateManager
	llmClient   *LLMClient
	sessionStore *SessionStore
}

func NewHandler(config *Config, stateMgr *StateManager, llmClient *LLMClient, sessionStore *SessionStore) *Handler {
	return &Handler{
		config:      config,
		stateMgr:    stateMgr,
		llmClient:   llmClient,
		sessionStore: sessionStore,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2 Proxy for LLM</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #333; margin-bottom: 20px; }
        .btn {
            display: inline-block;
            background: #0066cc;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            margin-top: 20px;
        }
        .btn:hover { background: #0055aa; }
        .info { color: #666; margin-top: 15px; font-size: 14px; }
    </style>
</head>
<body>
    <div class="card">
        <h1>🔐 OAuth2 Proxy</h1>
        <p>Proxy to your local LLM with OAuth2 authentication</p>
        <a href="/auth" class="btn">Login</a>
        <p class="info">API: <code>/v1/chat/completions</code></p>
    </div>
</body>
</html>
`))
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	state := h.stateMgr.GenerateState()
	
	authURL := h.config.AuthorizationEndpoint
	authURL += "?client_id=" + h.config.ClientID
	authURL += "&redirect_uri=" + h.config.RedirectURI
	authURL += "&response_type=code"
	authURL += "&state=" + state
	authURL += "&scope=" + strings.Join(h.config.Scopes, " ")
	
	http.Redirect(w, r, authURL, http.StatusFound)
	
	if r.URL.Query().Get("error") != "" {
		// Handle OAuth errors from the upstream provider
		http.Error(w, "OAuth error", http.StatusForbidden)
	}
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}
	
	redirectURI, err := h.stateMgr.ValidateState(state, h.config.RedirectURI)
	if err != nil {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	
	tokenReq := TokenRequest{
		GrantType:   "authorization_code",
		Code:        code,
		RedirectURI: redirectURI,
		ClientID:    h.config.ClientID,
		ClientSecret: h.config.ClientSecret,
	}
	
	tokenBody, err := json.Marshal(tokenReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}
	
	tokenURL := h.config.TokenEndpoint
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(string(tokenBody)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+
		btoa(h.config.ClientID+":"+h.config.ClientSecret))
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		http.Error(w, "Failed to decode token response", http.StatusInternalServerError)
		return
	}
	
	userInfo, err := h.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	
	session := &Session{
		Token:     tokenResp.AccessToken,
		UserInfo:  userInfo,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	
	h.sessionStore.Save(r, session)
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) getUserInfo(token string) (UserInfoResponse, error) {
	reqURL := h.config.UserInfoEndpoint
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return UserInfoResponse{}, err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UserInfoResponse{}, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return UserInfoResponse{}, err
	}
	
	var userInfo UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return UserInfoResponse{}, err
	}
	
	return userInfo, nil
}

func (h *Handler) Proxy(w http.ResponseWriter, r *http.Request) {
	session, exists := h.sessionStore.Get(r)
	if !exists {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}
	
	if time.Now().After(session.ExpiresAt) {
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}
	
	reqURL := h.config.LLMAPIURL + r.URL.Path
	reqURL += "?" + r.URL.RawQuery
	
	req, err := http.NewRequest(r.Method, reqURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	
	req.Header.Set("Authorization", "Bearer "+session.Token)
	for key, values := range r.Header {
		for _, value := range values {
			if key != "Host" {
				req.Header.Add(key, value)
			}
		}
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Proxy request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Failed to copy response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func btoa(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "+", "-"), "/", "_")
}
