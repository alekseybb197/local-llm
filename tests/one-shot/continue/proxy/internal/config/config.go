
package config

import (
	"os"
	"time"
)

type Config struct {
	Server ServerConfig
	OAuth  OAuthConfig
	LLM    LLMConfig
	JWT    JWTConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	RedirectURL  string
}

type LLMConfig struct {
	APIURL string
	// OpenAI-compatible endpoint prefix
	EndpointPrefix string
}

type JWTConfig struct {
	SecretKey     string
	Issuer        string
	Audience      string
	Expiration    time.Duration
	RefreshExp    time.Duration
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
		},
		OAuth: OAuthConfig{
			ClientID:     getEnv("OAUTH_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_CLIENT_SECRET", ""),
			TokenURL:     getEnv("OAUTH_TOKEN_URL", ""),
			RedirectURL:  getEnv("OAUTH_REDIRECT_URL", "http://localhost:8080/oauth/callback"),
		},
		LLM: LLMConfig{
			APIURL:          getEnv("LLM_API_URL", "http://localhost:11434/api"),
			EndpointPrefix:  getEnv("LLM_ENDPOINT_PREFIX", "/v1"),
		},
		JWT: JWTConfig{
			SecretKey:     getEnv("JWT_SECRET_KEY", "change-this-in-production"),
			Issuer:        getEnv("JWT_ISSUER", "oauth2-proxy"),
			Audience:      getEnv("JWT_AUDIENCE", "llm-api"),
			Expiration:    time.Hour * 24,
			RefreshExp:    time.Hour * 24 * 7,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
