package handlers

import (
	"encoding/json"
	"net/http"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	oauth2 *OAuth2Handler
	store  interface{}
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(store interface{}) *AuthHandler {
	return &AuthHandler{
		store: store,
	}
}

// Login initiates the OAuth2 login flow
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.oauth2 == nil {
		http.Error(w, "OAuth2 not configured", http.StatusInternalServerError)
		return
	}
	h.oauth2.Login(w, r)
}

// Callback handles the OAuth2 callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.oauth2 == nil {
		http.Error(w, "OAuth2 not configured", http.StatusInternalServerError)
		return
	}
	h.oauth2.Callback(w, r)
}

// Logout logs out the current user
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.oauth2 == nil {
		http.Error(w, "OAuth2 not configured", http.StatusInternalServerError)
		return
	}
	h.oauth2.Logout(w, r)
}

// LoginStatus checks if the user is logged in
func (h *AuthHandler) LoginStatus(w http.ResponseWriter, r *http.Request) {
	if h.oauth2 == nil {
		http.Error(w, "OAuth2 not configured", http.StatusInternalServerError)
		return
	}
	h.oauth2.LoginStatus(w, r)
}

// Me returns the current user information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if h.oauth2 == nil {
		http.Error(w, "OAuth2 not configured", http.StatusInternalServerError)
		return
	}
	// In a real implementation, this would use the session from OAuth2Handler
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": nil,
	})
}
