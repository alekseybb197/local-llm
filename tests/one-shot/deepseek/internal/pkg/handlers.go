package pkg

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Handler handles OAuth2 requests
type Handler struct {
	stateMachine *StateMachine
	config       Config
}

// NewHandler creates a new handler
func NewHandler(stateMachine *StateMachine, config Config) *Handler {
	return &Handler{
		stateMachine: stateMachine,
		config:       config,
	}
}

// StartAuthorization initiates OAuth2 authorization flow
func (h *Handler) StartAuthorization(w http.ResponseWriter, r *http.Request) {
	requestState := r.URL.Query().Get("state")
	redirectURL := r.URL.Query().Get("redirect")

	var authCode string
	var codeVerifier string
	var codeChallenge string

	// Generate PKCE code verifier and challenge if enabled
	if h.config.EnablePKCE {
		codeVerifier = generateCodeVerifier()
		codeChallenge = generateCodeChallenge(codeVerifier)
	}

	if authCode == "" {
		authCode = generateAuthCode()
	}

	// Create state
	state, err := NewState(authCode, codeVerifier, codeChallenge, redirectURL)
	if err != nil {
		http.Error(w, "Failed to create authorization state", http.StatusInternalServerError)
		return
	}

	// Store state
	h.stateMachine.StoreState(state)

	// Build authorization URL
	authorizationURL := fmt.Sprintf("%s/oauth2/auth", h.config.ProviderURL)
	if requestState != "" {
		authorizationURL += fmt.Sprintf("?state=%s", url.QueryEscape(requestState))
	}
	if redirectURL != "" {
		authorizationURL += fmt.Sprintf("&redirect=%s", url.QueryEscape(redirectURL))
	}

	// Add PKCE parameters if enabled
	if h.config.EnablePKCE {
		authorizationURL += fmt.Sprintf("&code_challenge=%s", url.QueryEscape(codeChallenge))
		authorizationURL += fmt.Sprintf("&code_challenge_method=S256")
	}

	http.Redirect(w, r, authorizationURL, http.StatusFound)
}

// Callback handles the OAuth2 callback from the provider
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	stateToken := r.URL.Query().Get("state")
	authCode := r.URL.Query().Get("auth_code")

	if authCode == "" {
		http.Error(w, "Missing auth_code", http.StatusBadRequest)
		return
	}

	// Retrieve state
	state, exists := h.stateMachine.GetState(stateToken)
	if !exists {
		http.Error(w, "Invalid or expired state", http.StatusForbidden)
		return
	}

	// Validate state token
	if state.AuthCode != authCode {
		http.Error(w, "State mismatch", http.StatusForbidden)
		return
	}

	// Exchange authorization code for access token
	accessToken, err := h.exchangeCode(authCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete state after use
	h.stateMachine.DeleteState(stateToken)

	// Redirect to client with token
	redirectURL := h.config.RedirectURL
	if state.RedirectURL != "" {
		redirectURL = state.RedirectURL
	}

	redirectParams := map[string]string{
		"access_token": accessToken.Token,
		"token_type":   accessToken.TokenType,
		"expires_in":   strconv.FormatInt(int64(accessToken.ExpiresIn), 10),
	}

	if state.CodeVerifier != "" {
		redirectParams["code_verifier"] = state.CodeVerifier
	}

	redirectURL += "?"
	for k, v := range redirectParams {
		redirectURL += fmt.Sprintf("%s=%s&", k, url.QueryEscape(v))
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// TokenEndpoint exchanges authorization code for access token
func (h *Handler) TokenEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	clientSecret := r.FormValue("client_secret")
	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")

	if grantType != "authorization_code" {
		http.Error(w, "Unsupported grant type", http.StatusUnsupportedMediaType)
		return
	}

	if code == "" {
		http.Error(w, "Authorization code required", http.StatusBadRequest)
		return
	}

	// Verify client secret
	if clientSecret != h.config.ClientSecret {
		http.Error(w, "Invalid client secret", http.StatusUnauthorized)
		return
	}

	// Exchange code for token
	accessToken, err := h.exchangeCode(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
		return
	}

	// Return token response
	response := map[string]string{
		"access_token": accessToken.Token,
		"token_type":   accessToken.TokenType,
		"expires_in":   strconv.FormatInt(int64(accessToken.ExpiresIn), 10),
	}

	if accessToken.RefreshToken != "" {
		response["refresh_token"] = accessToken.RefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// InfoEndpoint returns OAuth2 server information
func (h *Handler) InfoEndpoint(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"authorization_endpoint": fmt.Sprintf("%s/oauth2/auth", h.config.ProviderURL),
		"token_endpoint":         fmt.Sprintf("%s/oauth2/token", h.config.ProviderURL),
		"issuer":                 h.config.ProviderURL,
		"grant_types":            []string{"authorization_code"},
		"scopes":                 h.config.Scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// RefreshEndpoint exchanges refresh token for new access token
func (h *Handler) RefreshEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	clientSecret := r.FormValue("client_secret")
	grantType := r.FormValue("grant_type")
	refreshToken := r.FormValue("refresh_token")

	if grantType != "refresh_token" {
		http.Error(w, "Unsupported grant type", http.StatusUnsupportedMediaType)
		return
	}

	if refreshToken == "" {
		http.Error(w, "Refresh token required", http.StatusBadRequest)
		return
	}

	// Verify client secret
	if clientSecret != h.config.ClientSecret {
		http.Error(w, "Invalid client secret", http.StatusUnauthorized)
		return
	}

	// Exchange refresh token for new access token
	newAccessToken := h.exchangeRefreshToken(refreshToken)

	// Return token response
	response := map[string]string{
		"access_token": newAccessToken.Token,
		"token_type":   newAccessToken.TokenType,
		"expires_in":   strconv.FormatInt(int64(newAccessToken.ExpiresIn), 10),
	}

	if newAccessToken.RefreshToken != "" {
		response["refresh_token"] = newAccessToken.RefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// exchangeCode exchanges authorization code for access token
func (h *Handler) exchangeCode(authCode string) (*AccessToken, error) {
	// In a real implementation, this would make a request to the provider's token endpoint
	// For now, we'll generate a mock token
	return &AccessToken{
		Token:        "mock_access_token_" + authCode,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "mock_refresh_token_" + authCode,
	}, nil
}

// exchangeRefreshToken exchanges refresh token for new access token
func (h *Handler) exchangeRefreshToken(refreshToken string) *AccessToken {
	// In a real implementation, this would make a request to the provider's token endpoint
	// For now, we'll generate a mock token
	return &AccessToken{
		Token:        "mock_access_token_" + refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "mock_refresh_token_" + refreshToken,
	}
}

// generateCodeVerifier generates a random code verifier (43-128 characters)
func generateCodeVerifier() string {
	// Generate 32 bytes and base64 encode to get ~43 characters
	bytes := make([]byte, 32)
	randBytes := cryptoRandomBytes(32)
	bytes = append(bytes, randBytes...)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// generateCodeChallenge generates a code challenge from verifier
func generateCodeChallenge(codeVerifier string) string {
	// SHA256 of the verifier
	hash := hmac.New(sha256.New, nil)
	hash.Write([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

// generateAuthCode generates a mock authorization code
func generateAuthCode() string {
	return "mock_auth_code_" + time.Now().Format(time.RFC3339)
}

// cryptoRandomBytes returns random bytes
func cryptoRandomBytes(n int) []byte {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return bytes
}

// AccessToken represents an OAuth2 access token
type AccessToken struct {
	Token        string
	TokenType    string
	ExpiresIn    int64 // seconds
	RefreshToken string
}

// RefreshToken represents OAuth2 refresh token
type RefreshToken struct {
	RefreshToken string
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	RefreshToken `json:"refresh_token,omitempty"`
}
