package models

import (
	"time"
)

type Client struct {
	ID             string    `json:"id"`
	ClientID       string    `json:"client_id"`
	ClientSecret   string    `json:"client_secret"`
	RedirectURI    string    `json:"redirect_uri"`
	Scopes         []string  `json:"scopes"`
	GrantTypes     []string  `json:"grant_types"`
	CreatedAt      time.Time `json:"created_at"`
}

type Token struct {
	ID             string    `json:"id"`
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	ExpiresAt      time.Time `json:"expires_at"`
	TokenType      string    `json:"token_type"`
	Scopes         string    `json:"scopes"`
	Subject        string    `json:"subject"`
	ClientID       string    `json:"client_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type OAuthCode struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	ClientID       string    `json:"client_id"`
	RedirectURI    string    `json:"redirect_uri"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	Scopes         string    `json:"scopes"`
}

type PKCECodeVerifier struct {
	ID                        string    `json:"id"`
	CodeVerifier              string    `json:"code_verifier"`
	CodeChallenge             string    `json:"code_challenge"`
	CodeChallengeMethod       string    `json:"code_challenge_method"`
	CreatedAt                 time.Time `json:"created_at"`
}

type AuthRequest struct {
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	ResponseType string `json:"response_type"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	Prompt       string `json:"prompt"`
	AccessType   string `json:"access_type"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope       string `json:"scope"`
}

type TokenRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code"`
	ClientID    string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	CodeVerifier string `json:"code_verifier"`
	RefreshToken string `json:"refresh_token"`
	RedirectURI string `json:"redirect_uri"`
}

type ErrorResponse struct {
	Error              string `json:"error"`
	ErrorDescription   string `json:"error_description"`
	ErrorURI           string `json:"error_uri"`
}

type ChatCompletionRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Stream       bool      `json:"stream"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	TopP         float64   `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty float64 `json:"presence_penalty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type ChatCompletionResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Create            int64       `json:"created"`
	Model             string      `json:"model"`
	Choices           []Choice    `json:"choices"`
	Usage             Usage       `json:"usage"`
	SystemFingerprint string      `json:"system_fingerprint"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
