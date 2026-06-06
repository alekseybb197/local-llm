// Package config provides configuration management for the OAuth2 proxy.
package config

import (
"os"
"strconv"
"time"
)

// Config holds all configuration for the OAuth2 proxy.
type Config struct {
ServerConfig  ServerConfig
OAuth2Config  OAuth2Config
ProxyConfig   ProxyConfig
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
Host        string
Port        int
ReadTimeout time.Duration
WriteTimeout time.Duration
IdleTimeout time.Duration
}

// OAuth2Config holds OAuth2 provider configuration.
type OAuth2Config struct {
ClientID       string
ClientSecret   string
RedirectURI    string
Scope          string
TokenURL       string
AuthURL        string
UserInfoURL    string
}

// ProxyConfig holds proxy-specific configuration.
type ProxyConfig struct {
LLMEndpoint   string
APIKey        string
DefaultModel  string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
return loadFromEnv()
}

// loadFromEnv loads configuration from environment variables.
func loadFromEnv() (*Config, error) {
portStr := getEnv("SERVER_PORT", "8080")
port, err := strconv.Atoi(portStr)
if err != nil {
port = 8080
}

return &Config{
ServerConfig: ServerConfig{
Host:        getEnv("SERVER_HOST", "localhost"),
Port:        port,
ReadTimeout: durationFromEnv("SERVER_READ_TIMEOUT", 30*time.Second),
WriteTimeout: durationFromEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
IdleTimeout:  durationFromEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
},
OAuth2Config: OAuth2Config{
ClientID:       getEnv("OAUTH2_CLIENT_ID", ""),
ClientSecret:   getEnv("OAUTH2_CLIENT_SECRET", ""),
RedirectURI:    getEnv("OAUTH2_REDIRECT_URI", "http://localhost:8080/oauth2/callback"),
Scope:          getEnv("OAUTH2_SCOPE", "openid email profile"),
TokenURL:       getEnv("OAUTH2_TOKEN_URL", "https://example.com/oauth/token"),
AuthURL:        getEnv("OAUTH2_AUTH_URL", "https://example.com/oauth/authorize"),
UserInfoURL:    getEnv("OAUTH2_USER_INFO_URL", "https://example.com/oauth/userinfo"),
},
ProxyConfig: ProxyConfig{
LLMEndpoint:   getEnv("LLM_ENDPOINT", "http://localhost:11434"),
APIKey:        getEnv("LLM_API_KEY", ""),
DefaultModel:  getEnv("DEFAULT_MODEL", "llama3"),
},
}, nil
}

func getEnv(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

func durationFromEnv(key string, defaultValue time.Duration) time.Duration {
if value := os.Getenv(key); value != "" {
if d, err := time.ParseDuration(value); err == nil {
return d
}
}
return defaultValue
}
