package config

import (
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
)

type OAuth2Config struct {
	ClientID        string `mapstructure:"client_id"`
	ClientSecret    string `mapstructure:"client_secret"`
	RedirectURL     string `mapstructure:"redirect_url"`
	AuthorizationURL string `mapstructure:"authorization_url"`
	TokenURL        string `mapstructure:"token_url"`
	UserInfoURL     string `mapstructure:"user_info_url"`
}

type Config struct {
	// OAuth2 Configuration
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	// Authorization endpoint (e.g., http://localhost:8081/oauth/authorize)
	AuthorizationURL string `mapstructure:"authorization_url"`
	// Token endpoint (e.g., http://localhost:8081/oauth/token)
	TokenURL string `mapstructure:"token_url"`
	// UserInfo endpoint (e.g., http://localhost:8081/oauth/userinfo)
	UserInfoURL string `mapstructure:"user_info_url"`

	// LLM Proxy Configuration
	ListenAddr     string        `mapstructure:"listen_addr"`
	LLMAPIURL      string        `mapstructure:"llm_api_url"`
	LLMHeaders     string        `mapstructure:"llm_headers"`
	Timeout        time.Duration `mapstructure:"timeout"`
	Certificate    string        `mapstructure:"cert"`
	Key            string        `mapstructure:"key"`

	// Storage Configuration
	StorageDriver string `mapstructure:"storage_driver"`
	StoragePath   string `mapstructure:"storage_path"`

	// CORS Configuration
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

func DefaultConfig() *Config {
	return &Config{
		ListenAddr:     ":8080",
		LLMAPIURL:      "http://localhost:11434/api/generate",
		Timeout:        30 * time.Second,
		StorageDriver:  "memory",
		AllowMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:   []string{"Authorization", "Content-Type", "Accept", "Origin"},
	}
}

func Load(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := mapstructure.Decode(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) GetOAuth2Config() *OAuth2Config {
	return &OAuth2Config{
		ClientID:        c.ClientID,
		ClientSecret:    c.ClientSecret,
		RedirectURL:     c.RedirectURL,
		AuthorizationURL: c.AuthorizationURL,
		TokenURL:        c.TokenURL,
		UserInfoURL:     c.UserInfoURL,
	}
}
