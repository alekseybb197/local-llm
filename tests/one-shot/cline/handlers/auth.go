package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/oauth2-proxy/local-llm-proxy/config"
	"github.com/oauth2-proxy/local-llm-proxy/db"
	"github.com/oauth2-proxy/local-llm-proxy/models"
	"github.com/oauth2-proxy/local-llm-proxy/oauth"
)

// TokenHandler provides public handler functions for routing
type TokenHandler struct {
	authHandler *AuthHandler
}

func NewTokenHandler(database *db.Database, oauthServer *oauth.OAuthServer, cfg *config.Config) *TokenHandler {
	authHandler := NewAuthHandler(database, oauthServer, cfg)
	return &TokenHandler{authHandler: authHandler}
}

func (h *TokenHandler) Token(w http.ResponseWriter, r *http.Request) {
	h.authHandler.Token(w, r)
}

func (h *TokenHandler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	h.authHandler.RegisterClient(w, r)
}

func (h *TokenHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	h.authHandler.DeleteClient(w, r)
}

func (h *TokenHandler) ClientInfo(w http.ResponseWriter, r *http.Request) {
	h.authHandler.ClientInfo(w, r)
}

// AuthHandler handles OAuth operations
type AuthHandler struct {
	database      *db.Database
	oauthServer   *oauth.OAuthServer
	config        *config.Config
	clientSecrets map[string]string
}

func NewAuthHandler(database *db.Database, oauthServer *oauth.OAuthServer, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		database:      database,
		oauthServer:   oauthServer,
		config:        cfg,
		clientSecrets: make(map[string]string),
	}
}

func (h *AuthHandler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.Client
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		log.Printf("Failed to decode client registration: %v", err)
		return
	}

	// Validate required fields
	if req.ClientID == "" || req.ClientSecret == "" || req.RedirectURI == "" {
		http.Error(w, `{"error": "Missing required fields: client_id, client_secret, redirect_uri"}`, http.StatusBadRequest)
		return
	}

	// Create client
	client := &models.Client{
		ID:          time.Now().Format("20060102150405") + "-" + base64.StdEncoding.EncodeToString([]byte(req.ClientID)),
		ClientID:    req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURI:  req.RedirectURI,
		Scopes:       req.Scopes,
		GrantTypes:   req.GrantTypes,
		CreatedAt:    time.Now(),
	}

	if err := h.database.CreateClient(client); err != nil {
		http.Error(w, `{"error": "Failed to create client"}`, http.StatusInternalServerError)
		log.Printf("Failed to create client: %v", err)
		return
	}

	log.Printf("Registered new client: %s", req.ClientID)

	// Return client credentials
	resp := map[string]interface{}{
		"client_id":     req.ClientID,
		"client_secret": req.ClientSecret,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, `{"error": "Missing client_id parameter"}`, http.StatusBadRequest)
		return
	}

	if err := h.database.DeleteClient(clientID); err != nil {
		http.Error(w, `{"error": "Failed to delete client"}`, http.StatusInternalServerError)
		log.Printf("Failed to delete client: %v", err)
		return
	}

	log.Printf("Deleted client: %s", clientID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Client deleted successfully"})
}

func (h *AuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	h.oauthServer.Authorize(w, r)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	h.oauthServer.Callback(w, r)
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req models.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		log.Printf("Failed to decode token request: %v", err)
		return
	}

	// Handle authorization code grant
	if req.GrantType == "authorization_code" {
		h.handleAuthorizationCode(w, r, &req)
		return
	}

	// Handle refresh token grant
	if req.GrantType == "refresh_token" {
		h.handleRefreshToken(w, r, &req)
		return
	}

	http.Error(w, `{"error": "Invalid grant_type"}`, http.StatusBadRequest)
}

func (h *AuthHandler) handleAuthorizationCode(w http.ResponseWriter, r *http.Request, req *models.TokenRequest) {
	code := req.Code
	clientID := req.ClientID
	clientSecret := req.ClientSecret

	// Get client from database
	client, err := h.database.GetClient(clientID)
	if err != nil {
		http.Error(w, `{"error": "Invalid client_id"}`, http.StatusBadRequest)
		log.Printf("Failed to get client: %v", err)
		return
	}

	// Verify client secret
	if client.ClientSecret != clientSecret {
		http.Error(w, `{"error": "Invalid client_secret"}`, http.StatusUnauthorized)
		return
	}

	// Get authorization code
	codeObj, err := h.database.GetCodeByCode(code)
	if err != nil {
		http.Error(w, `{"error": "Invalid authorization code"}`, http.StatusBadRequest)
		log.Printf("Failed to get authorization code: %v", err)
		return
	}

	// Check if code is expired
	if expired, err := h.database.IsCodeExpiredByCode(code); err != nil {
		http.Error(w, `{"error": "Failed to check code expiration"}`, http.StatusInternalServerError)
		log.Printf("Failed to check code expiration: %v", err)
		return
	} else if expired {
		http.Error(w, `{"error": "Authorization code expired"}`, http.StatusUnauthorized)
		return
	}

	// Create access token
	accessToken := code
	token := &models.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(3600 * time.Second), // 1 hour
		RefreshToken: "",
		Scopes:       codeObj.Scopes,
		Subject:      "user",
		ClientID:     clientID,
		CreatedAt:    time.Now(),
	}

	if err := h.database.CreateToken(token); err != nil {
		http.Error(w, `{"error": "Failed to create token"}`, http.StatusInternalServerError)
		log.Printf("Failed to create token: %v", err)
		return
	}

	// Delete the used code
	if err := h.database.DeleteCodeByCode(code); err != nil {
		log.Printf("Failed to delete code: %v", err)
	}

	// Return response
	resp := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	if codeObj.Scopes != "" {
		resp["scope"] = codeObj.Scopes
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request, req *models.TokenRequest) {
	refreshToken := req.RefreshToken
	clientID := req.ClientID
	clientSecret := req.ClientSecret

	// Get client from database
	client, err := h.database.GetClient(clientID)
	if err != nil {
		http.Error(w, `{"error": "Invalid client_id"}`, http.StatusBadRequest)
		log.Printf("Failed to get client: %v", err)
		return
	}

	// Verify client secret
	if client.ClientSecret != clientSecret {
		http.Error(w, `{"error": "Invalid client_secret"}`, http.StatusUnauthorized)
		return
	}

	// Get existing token
	token, err := h.database.GetTokenByAccessToken(refreshToken)
	if err != nil {
		http.Error(w, `{"error": "Invalid refresh token"}`, http.StatusBadRequest)
		log.Printf("Failed to get token by refresh token: %v", err)
		return
	}

	// Create new access token
	newAccessToken := "new_access_token_" + time.Now().Format(time.RFC3339)
	newToken := &models.Token{
		AccessToken:  newAccessToken,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(3600 * time.Second),
		RefreshToken: refreshToken,
		Scopes:       token.Scopes,
		Subject:      token.Subject,
		ClientID:     clientID,
		CreatedAt:    time.Now(),
	}

	if err := h.database.CreateToken(newToken); err != nil {
		http.Error(w, `{"error": "Failed to create token"}`, http.StatusInternalServerError)
		log.Printf("Failed to create token: %v", err)
		return
	}

	// Return response
	resp := map[string]interface{}{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"refresh_token": refreshToken,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) ClientInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, `{"error": "Missing client_id parameter"}`, http.StatusBadRequest)
		return
	}

	client, err := h.database.GetClient(clientID)
	if err != nil {
		http.Error(w, `{"error": "Client not found"}`, http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"client_id":     client.ClientID,
		"redirect_uri":  client.RedirectURI,
		"scopes":        client.Scopes,
		"grant_types":   client.GrantTypes,
		"created_at":    client.CreatedAt.Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
