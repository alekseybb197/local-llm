package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ListenAddr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.ListenAddr)
	}
	if cfg.LLMAPIURL != "http://localhost:11434/api/generate" {
		t.Errorf("expected http://localhost:11434/api/generate, got %s", cfg.LLMAPIURL)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", cfg.Timeout)
	}
}

func TestOAuth2Config(t *testing.T) {
	cfg := &Config{
		ClientID:        "client-id",
		ClientSecret:    "secret",
		RedirectURL:     "http://localhost:8080/callback",
		AuthorizationURL: "http://localhost:8081/authorize",
		TokenURL:        "http://localhost:8081/token",
		UserInfoURL:     "http://localhost:8081/userinfo",
	}

	oauth2Cfg := cfg.GetOAuth2Config()

	if oauth2Cfg.ClientID != "client-id" {
		t.Errorf("expected client-id, got %s", oauth2Cfg.ClientID)
	}
	if oauth2Cfg.RedirectURL != "http://localhost:8080/callback" {
		t.Errorf("expected callback URL, got %s", oauth2Cfg.RedirectURL)
	}
}
