package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// OAuth2Config holds OAuth2 provider configuration
type OAuth2Config struct {
	// RedirectURI is the URL to which the authorization server will redirect the user after authorization
	RedirectURI string `json:"redirect_uri"`
	
	// ClientID is the OAuth2 client ID
	ClientID string `json:"client_id"`
	
	// ClientSecret is the OAuth2 client secret
	ClientSecret string `json:"client_secret"`
	
	// Scopes are the OAuth2 scopes to request
	Scopes []string `json:"scopes"`
	
	// EndpointURL is the authorization endpoint URL
	EndpointURL string `json:"endpoint_url"`
	
	// TokenURL is the token endpoint URL
	TokenURL string `json:"token_url"`
	
	// UserInfoURL is the user info endpoint URL
	UserInfoURL string `json:"user_info_url"`
	
	// SessionSecret is the secret for signing sessions
	SessionSecret string `json:"session_secret"`
	
	// Server is the server configuration (embedded for convenience)
	Server ServerConfig `json:"server,omitempty"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	// Host is the server host
	Host string `json:"host"`
	
	// Port is the server port
	Port int `json:"port"`
	
	// LLMAPIURL is the URL of the LLM API (OpenAI-compatible)
	LLMAPIURL string `json:"llm_api_url"`
	
	// LLMModel is the model name
	LLMModel string `json:"llm_model"`
	
	// AllowedOrigins is a list of allowed CORS origins
	AllowedOrigins []string `json:"allowed_origins"`
	
	// SessionSecret is the secret for signing sessions
	SessionSecret string `json:"session_secret"`
	
	// CookieExpiration is the cookie expiration duration
	CookieExpiration time.Duration `json:"cookie_expiration"`
	
	// EnableHTTPS indicates if HTTPS should be enabled
	EnableHTTPS bool `json:"enable_https"`
	
	// HTTPSKeyPath is the path to the TLS private key file
	HTTPSKeyPath string `json:"https_key_path"`
	
	// HTTPSCertPath is the path to the TLS certificate file
	HTTPSCertPath string `json:"https_cert_path"`
}

// StoreConfig holds store configuration
type StoreConfig struct {
	// Type is the store type (memory, redis, etc.)
	Type string `json:"type"`
	
	// MemoryEnabled enables in-memory storage
	MemoryEnabled bool `json:"memory_enabled"`
}

// Config holds all configuration
type Config struct {
	// AppName is the application name
	AppName string `json:"app_name"`
	
	// OAuth2 is the OAuth2 configuration
	OAuth2 OAuth2Config `json:"oauth2"`
	
	// Server is the server configuration
	Server ServerConfig `json:"server"`
	
	// Store is the store configuration
	Store StoreConfig `json:"store"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		AppName: "hermes-oauth2-proxy",
		OAuth2: OAuth2Config{
			RedirectURI:  "",
			ClientID:     "",
			ClientSecret: "",
			Scopes:       []string{"openid", "profile", "email"},
			EndpointURL:  "",
			TokenURL:     "",
			UserInfoURL:  "",
		},
		Server: ServerConfig{
			Host:           "localhost",
			Port:           8080,
			LLMAPIURL:      "http://localhost:11434/api/generate",
			LLMModel:       "local-llm",
			AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
			SessionSecret:  "",
			CookieExpiration: 1 * time.Hour,
			EnableHTTPS:     false,
			HTTPSKeyPath:    "",
			HTTPSCertPath:   "",
		},
		Store: StoreConfig{
			Type:             "memory",
			MemoryEnabled:    true,
		},
	}
}

// parsePort parses a port from a string, allowing for optional "port:" prefix
func parsePort(s string) (int, error) {
	// Trim whitespace
	s = strings.TrimSpace(s)
	
	// Check for "port:" prefix
	if len(s) >= 6 && s[:5] == "port:" {
		s = strings.TrimSpace(s[5:])
	}
	
	return strconv.Atoi(s)
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	if path == "" {
		// If no path is provided, use default config with env overrides
		cfg := DefaultConfig()
		
		if v := os.Getenv("HERMES_APP_NAME"); v != "" {
			cfg.AppName = v
		}
		if v := os.Getenv("HERMES_OAUTH2_REDIRECT_URI"); v != "" {
			cfg.OAuth2.RedirectURI = v
		}
		if v := os.Getenv("HERMES_OAUTH2_CLIENT_ID"); v != "" {
			cfg.OAuth2.ClientID = v
		}
		if v := os.Getenv("HERMES_OAUTH2_CLIENT_SECRET"); v != "" {
			cfg.OAuth2.ClientSecret = v
		}
		if v := os.Getenv("HERMES_SERVER_HOST"); v != "" {
			cfg.Server.Host = v
		}
		if v := os.Getenv("HERMES_SERVER_PORT"); v != "" {
			port, err := parsePort(v)
			if err == nil {
				cfg.Server.Port = port
			}
		}
		if v := os.Getenv("HERMES_SERVER_SESSION_SECRET"); v != "" {
			cfg.Server.SessionSecret = v
		}

		return cfg, nil
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Allow overriding via environment variables
	if v := os.Getenv("HERMES_APP_NAME"); v != "" {
		cfg.AppName = v
	}
	if v := os.Getenv("HERMES_OAUTH2_REDIRECT_URI"); v != "" {
		cfg.OAuth2.RedirectURI = v
	}
	if v := os.Getenv("HERMES_OAUTH2_CLIENT_ID"); v != "" {
		cfg.OAuth2.ClientID = v
	}
	if v := os.Getenv("HERMES_OAUTH2_CLIENT_SECRET"); v != "" {
		cfg.OAuth2.ClientSecret = v
	}
	if v := os.Getenv("HERMES_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("HERMES_SERVER_PORT"); v != "" {
		port, err := parsePort(v)
		if err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("HERMES_SERVER_SESSION_SECRET"); v != "" {
		cfg.Server.SessionSecret = v
	}

	return &cfg, nil
}
