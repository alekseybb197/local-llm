package models

import (
	"encoding/json"
	"time"
)

const (
	ClientID     = "oauth2-proxy-client"
	ClientSecret = "super-secret-client-key-change-in-production"
	RedirectURI  = "http://localhost:8080/callback"
)

type User struct {
	ID           string    `json:"id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type OAuth2Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scopes       string `json:"scope"`
}

type State struct {
	CodeVerifier string
	StateToken    string
	CodeChallenge string
}

type Session struct {
	StateToken string    `json:"state_token"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type StoredSession struct {
	StateToken string    `json:"state_token"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *StoredSession) ToSession() *Session {
	return &Session{
		StateToken: s.StateToken,
		UserID:     s.UserID,
		CreatedAt:  s.CreatedAt,
	}
}

func (s *StoredSession) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		StateToken string    `json:"state_token"`
		UserID     string    `json:"user_id"`
		CreatedAt  time.Time `json:"created_at"`
	}{
		StateToken: s.StateToken,
		UserID:     s.UserID,
		CreatedAt:  s.CreatedAt,
	})
}

func (s *StoredSession) UnmarshalJSON(data []byte) error {
	var tmp struct {
		StateToken string    `json:"state_token"`
		UserID     string    `json:"user_id"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	s.StateToken = tmp.StateToken
	s.UserID = tmp.UserID
	s.CreatedAt = tmp.CreatedAt
	return nil
}
