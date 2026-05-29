package main

import (
	"os"
	"time"
)

type Config struct {
	ServerPort           string
	LLMAPIURL            string
	LLMAPIKey            string
	ClientID             string
	ClientSecret         string
	RedirectURI          string
	AuthorizationEndpoint string
	TokenEndpoint        string
	UserInfoEndpoint     string
	Scopes               []string
	StateSecret          string
	CookieName           string
	SessionTimeout       time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		ServerPort:           "8080",
		LLMAPIURL:            os.Getenv("LLM_API_URL") + "/v1",
		LLMAPIKey:            os.Getenv("LLM_API_KEY"),
		ClientID:             os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret:         os.Getenv("OAUTH_CLIENT_SECRET"),
		RedirectURI:          os.Getenv("OAUTH_REDIRECT_URI"),
		AuthorizationEndpoint: os.Getenv("OAUTH_AUTH_ENDPOINT"),
		TokenEndpoint:        os.Getenv("OAUTH_TOKEN_ENDPOINT"),
		UserInfoEndpoint:     os.Getenv("OAUTH_USER_INFO_ENDPOINT"),
		Scopes:               []string{"openid", "profile", "email"},
		StateSecret:          os.Getenv("OAUTH_STATE_SECRET"),
		CookieName:           "oauth_state",
		SessionTimeout:       10 * time.Minute,
	}
}

func (c *Config) Validate() error {
	if c.ClientID == "" {
		return ErrMissingClientID
	}
	if c.ClientSecret == "" {
		return ErrMissingClientSecret
	}
	if c.RedirectURI == "" {
		return ErrMissingRedirectURI
	}
	if c.AuthorizationEndpoint == "" {
		return ErrMissingAuthEndpoint
	}
	if c.TokenEndpoint == "" {
		return ErrMissingTokenEndpoint
	}
	if c.StateSecret == "" {
		return ErrMissingStateSecret
	}
	return nil
}
