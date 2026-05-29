package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	hermes "hermes/internal/store"
	hermesconfig "hermes/config"
	"sync"
)

// OAuth2Handler handles OAuth2 Authorization Code Flow
type OAuth2Handler struct {
	config      *hermesconfig.OAuth2Config
	store       hermes.Store
	httpClient  *http.Client
	storeMutex  sync.Mutex
}

// NewOAuth2Handler creates a new OAuth2 handler
func NewOAuth2Handler(cfg *hermesconfig.OAuth2Config, store hermes.Store) *OAuth2Handler {
	return &OAuth2Handler{
		config:      cfg,
		store:       store,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Login initiates the OAuth2 authorization flow
func (h *OAuth2Handler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := hermes.GenerateState()
	if err != nil {
		http.Error(w, "Failed to generate state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	stateObj := &hermes.State{
		State:     state,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := h.store.SaveState(context.Background(), stateObj); err != nil {
		http.Error(w, "Failed to save state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	authURL := h.buildAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the OAuth2 callback from the authorization server
func (h *OAuth2Handler) Callback(w http.ResponseWriter, r *http.Request) {
	stateParam := r.URL.Query().Get("state")
	if stateParam == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	st, err := h.store.GetState(context.Background(), stateParam)
	if err != nil {
		http.Error(w, "Invalid state: "+err.Error(), http.StatusBadRequest)
		return
	}

	if time.Now().After(st.ExpiresAt) {
		http.Error(w, "State has expired", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	token, err := h.exchangeToken(code, st.State)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusBadRequest)
		return
	}

	userInfo, err := h.getUserInfo(token)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.SaveTokenAndUserInfo(context.Background(), st.State, token, userInfo); err != nil {
		http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session := &Session{
		UserInfo: userInfo,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := h.createSession(w, session); err != nil {
		http.Error(w, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	successURL := r.URL.Query().Get("redirect_uri")
	if successURL == "" {
		successURL = "/"
	}
	http.Redirect(w, r, successURL, http.StatusFound)
}

func (h *OAuth2Handler) buildAuthURL(state string) string {
	oauth2Config := &oauth2.Config{
		ClientID:     h.config.ClientID,
		ClientSecret: h.config.ClientSecret,
		RedirectURL:  h.config.RedirectURI,
		Scopes:       h.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  h.config.EndpointURL,
			TokenURL: h.config.TokenURL,
		},
	}

	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url
}

func (h *OAuth2Handler) exchangeToken(code, state string) (*oauth2.Token, error) {
	oauth2Config := &oauth2.Config{
		ClientID:     h.config.ClientID,
		ClientSecret: h.config.ClientSecret,
		RedirectURL:  h.config.RedirectURI,
		Scopes:       h.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  h.config.EndpointURL,
			TokenURL: h.config.TokenURL,
		},
	}

	token, err := oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("oauth2: %w", err)
	}

	if token == nil {
		return nil, fmt.Errorf("token is nil")
	}

	return token, nil
}

func (h *OAuth2Handler) getUserInfo(token *oauth2.Token) (map[string]interface{}, error) {
	oauth2Config := &oauth2.Config{
		ClientID:     h.config.ClientID,
		ClientSecret: h.config.ClientSecret,
		RedirectURL:  h.config.RedirectURI,
		Scopes:       h.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  h.config.EndpointURL,
			TokenURL: h.config.TokenURL,
		},
	}

	tokenSource := oauth2Config.TokenSource(context.Background(), token)

	client := h.httpClient
	client.Transport = &oauth2.Transport{
		Source: tokenSource,
		Base:   http.DefaultTransport,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", h.config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return userInfo, nil
}

// Logout logs out the current user
func (h *OAuth2Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.deleteSession(w); err != nil {
		http.Error(w, "Failed to delete session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// LoginStatus checks if the user is logged in
func (h *OAuth2Handler) LoginStatus(w http.ResponseWriter, r *http.Request) {
	session, err := h.getSession(r)
	if err != nil || session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"loggedIn": false,
			"user":     nil,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	userData := session.UserInfo
	if userData == nil {
		userData = map[string]interface{}{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"loggedIn": true,
		"user":     userData,
	})
}

// Session holds session data
type Session struct {
	UserInfo map[string]interface{} `json:"user_info,omitempty"`
	Token     *oauth2.Token          `json:"token,omitempty"`
	CreatedAt time.Time              `json:"created_at,omitempty"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
}

func (h *OAuth2Handler) createSession(w http.ResponseWriter, session *Session) error {
	if h.config.SessionSecret == "" {
		return fmt.Errorf("session secret not configured")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "hermes_session",
		Value:    base64Encode(data),
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   h.config.Server.EnableHTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (h *OAuth2Handler) getSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("hermes_session")
	if err != nil {
		return nil, nil
	}

	data := base64Decode(cookie.Value)
	if data == nil {
		return nil, fmt.Errorf("invalid session data")
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (h *OAuth2Handler) deleteSession(w http.ResponseWriter) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "hermes_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.config.Server.EnableHTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(data string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	return decoded
}
