package models

import (
	"time"
)

type OAuth2AuthCode struct {
	Code          string
	ClientID      string
	Scopes        string
	RedirectURI   string
	Nonce         string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Used          bool
}

type OAuth2Token struct {
	AccessToken   string
	RefreshToken  string
	ExpiresAt     time.Time
	Scopes        string
	Code          string
	ClientID      string
	UserInfo      *UserInfo
	Nonce         string
	CreatedAt     time.Time
}

type UserInfo struct {
	ID          string
	Email       string
	DisplayName string
	Picture     string
	Profile     string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type UserInfoResponse struct {
	Sub        string `json:"sub"`
	Email      string `json:"email"`
	EmailVerified bool `json:"email_verified"`
	DisplayName string `json:"name"`
	Picture    string `json:"picture"`
}
