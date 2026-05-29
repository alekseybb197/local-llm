// Package oauth2 implements OAuth2 Authorization Code Flow.
package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openhands/oauth2-proxy/pkg/config"
)

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type UserInfo struct {
	ID    string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type State struct {
	Code       string
	State      string
	RedirectTo string
	Timestamp  time.Time
}

type Manager struct {
	states map[string]*State
}

func NewManager() *Manager {
	return &Manager{
		states: make(map[string]*State),
	}
}

func (m *Manager) GenerateState() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:32]
}

func (m *Manager) ValidateState(state string) bool {
	s, exists := m.states[state]
	if !exists {
		return false
	}
	if time.Since(s.Timestamp) > 5*time.Minute {
		delete(m.states, state)
		return false
	}
	delete(m.states, state)
	return true
}

func (m *Manager) SaveState(state string, code string, redirect string) {
	m.states[state] = &State{
		Code:       code,
		State:      state,
		RedirectTo: redirect,
		Timestamp:  time.Now(),
	}
}

func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

func RequestToken(cfg config.OAuth2Config, code string, redirectURI string) (*Token, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client credentials not configured")
	}
	data := url.Values{
		"grant_type":  {"authorization_code"},
		"client_id":   {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":        {code},
		"redirect_uri": {redirectURI},
	}
	req, err := http.NewRequest("POST", cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}
	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	return &token, nil
}

func RequestUserInfo(cfg config.OAuth2Config, accessToken string) (*UserInfo, error) {
	if cfg.UserInfoURL == "" {
		return nil, fmt.Errorf("userinfo URL not configured")
	}
	req, err := http.NewRequest("GET", cfg.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request user info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %w", err)
	}
	return &userInfo, nil
}

func generateStateParam(redirectURI string) string {
	state := generateSecureRandomString(32)
	type stateParam struct {
		State      string
		Timestamp  string
		RedirectTo string
	}
	now := time.Now().UTC().Format(time.RFC3339)
	data := stateParam{State: state, Timestamp: now, RedirectTo: redirectURI}
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func verifyStateParam(authCode string) error {
	data, err := base64.StdEncoding.DecodeString(authCode)
	if err != nil {
		return fmt.Errorf("failed to decode state: %w", err)
	}
	hash := sha256.Sum256(data)
	computedHash := sha256.Sum256(data)
	if !constantTimeCompare(hash[:], computedHash[:]) {
		return fmt.Errorf("state verification failed")
	}
	return nil
}

func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	x := make([]byte, len(a))
	for i := range a {
		x[i] = a[i] ^ b[i]
	}
	for _, v := range x {
		if v != 0 {
			return false
		}
	}
	return true
}
