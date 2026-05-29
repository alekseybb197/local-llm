package pkg

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	ServerPort        string
	ProviderURL       string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	Scopes            []string
	EnablePKCE        bool
	AuthCodeLifetime  time.Duration
	CORSAllowedOrigins []string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ServerPort:        ":8080",
		ProviderURL:       "http://localhost:8081",
		ClientID:          "",
		ClientSecret:      "",
		RedirectURL:       "",
		Scopes:            []string{"openid", "profile", "email"},
		EnablePKCE:        true,
		AuthCodeLifetime:  10 * time.Minute,
		CORSAllowedOrigins: []string{"*"},
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	config := DefaultConfig()

	if port := os.Getenv("PORT"); port != "" {
		config.ServerPort = port
	}

	if providerURL := os.Getenv("PROVIDER_URL"); providerURL != "" {
		config.ProviderURL = providerURL
	}

	if clientID := os.Getenv("CLIENT_ID"); clientID != "" {
		config.ClientID = clientID
	}

	if clientSecret := os.Getenv("CLIENT_SECRET"); clientSecret != "" {
		config.ClientSecret = clientSecret
	}

	if redirectURL := os.Getenv("REDIRECT_URL"); redirectURL != "" {
		config.RedirectURL = redirectURL
	}

	if scopes := os.Getenv("SCOPES"); scopes != "" {
		config.Scopes = splitCSV(scopes)
	}

	if pkce := os.Getenv("ENABLE_PKCE"); pkce != "" {
		if enabled, err := strconv.ParseBool(pkce); err == nil {
			config.EnablePKCE = enabled
		}
	}

	if authCodeLifetime := os.Getenv("AUTH_CODE_LIFETIME"); authCodeLifetime != "" {
		if duration, err := time.ParseDuration(authCodeLifetime); err == nil {
			config.AuthCodeLifetime = duration
		}
	}

	if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
		config.CORSAllowedOrigins = splitCSV(corsOrigins)
	}

	return config
}

// splitCSV splits a comma-separated value string
func splitCSV(s string) []string {
	result := make([]string, 0)
	for _, item := range strings.Split(s, ",") {
		if item != "" {
			result = append(result, strings.TrimSpace(item))
		}
	}
	return result
}
