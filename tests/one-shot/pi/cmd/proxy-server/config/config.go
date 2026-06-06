package config

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	HTTPAddr        string
	LLMProxyURL     string
	OAuthURL        string
	OAuthCallbackURL string
	OAuthClientID   string
	OAuthClientSecret string
	DBPath          string
	CORSOrigins     []string
}

func NewConfig(
	configFile, httpAddr, llmProxyURL, oauthURL, oauthCallbackURL,
	oauthClientID, oauthClientSecret, dbPath, corsOrigins string,
) *Config {
	return &Config{
		HTTPAddr:         httpAddr,
		LLMProxyURL:      llmProxyURL,
		OAuthURL:         oauthURL,
		OAuthCallbackURL: oauthCallbackURL,
		OAuthClientID:    oauthClientID,
		OAuthClientSecret: oauthClientSecret,
		DBPath:           dbPath,
		CORSOrigins:      parseOrigins(corsOrigins),
	}
}

func (c *Config) Validate() error {
	// Validate HTTP address
	if _, err := url.Parse(c.HTTPAddr); err != nil {
		return fmt.Errorf("invalid HTTP address: %w", err)
	}

	// Validate LLM proxy URL
	if c.LLMProxyURL == "" {
		return fmt.Errorf("LLM proxy URL is required")
	}
	if _, err := url.Parse(c.LLMProxyURL); err != nil {
		return fmt.Errorf("invalid LLM proxy URL: %w", err)
	}

	// Validate OAuth URL
	if c.OAuthURL == "" {
		return fmt.Errorf("OAuth URL is required")
	}
	if _, err := url.Parse(c.OAuthURL); err != nil {
		return fmt.Errorf("invalid OAuth URL: %w", err)
	}

	// Validate OAuth callback URL
	if c.OAuthCallbackURL == "" {
		return fmt.Errorf("OAuth callback URL is required")
	}
	if _, err := url.Parse(c.OAuthCallbackURL); err != nil {
		return fmt.Errorf("invalid OAuth callback URL: %w", err)
	}

	// Validate client credentials
	if c.OAuthClientID == "" {
		return fmt.Errorf("OAuth client ID is required")
	}
	if c.OAuthClientSecret == "" {
		return fmt.Errorf("OAuth client secret is required")
	}

	return nil
}

func parseOrigins(originsStr string) []string {
	if originsStr == "" {
		return nil
	}
	origins := strings.Split(strings.TrimSpace(originsStr), ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}
