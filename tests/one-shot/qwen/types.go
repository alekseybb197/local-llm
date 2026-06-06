package main

import (
	"encoding/json"
	"time"
)

type TokenResponse struct {
	AccessToken string            `json:"access_token"`
	TokenType   string            `json:"token_type"`
	ExpiresIn   int64             `json:"expires_in"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	Scopes      []string          `json:"scope,omitempty"`
	UserInfo    UserInfoResponse  `json:"user_info,omitempty"`
}

type UserInfoResponse struct {
	Sub   string `json:"sub"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type AuthRequest struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	ResponseType string `json:"response_type"`
	State       string `json:"state"`
	Scope       string `json:"scope,omitempty"`
}

type AuthResponse struct {
	Code    string `json:"code"`
	State   string `json:"state"`
}

type TokenRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	ClientID    string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type AuthState struct {
	State      string
	RedirectURI string
	Time       time.Time
}

type Session struct {
	Token     string
	UserInfo  UserInfoResponse
	ExpiresAt time.Time
}

type LLMRequest struct {
	Model  string `json:"model"`
	Messages []Message `json:"messages"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens int `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage `json:"usage"`
}

type Choice struct {
	Index     int     `json:"index"`
	Message   Message `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ErrorResponse struct {
	Error    string `json:"error"`
	ErrorDesc string `json:"error_description,omitempty"`
}

func (e *ErrorResponse) MarshalJSON() ([]byte, error) {
	type Alias ErrorResponse
	return json.Marshal(&struct {
		Error    string `json:"error"`
		ErrorDesc string `json:"error_description"`
	}{
		Error:    e.Error,
		ErrorDesc: e.ErrorDesc,
	})
}
