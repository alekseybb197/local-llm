package tests

import (
	"os"
	"path/filepath"
	"testing"

	"oauth2-proxy/cmd/proxy-server/db"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name      string
		origins   string
		expected  int
		hasError  bool
	}{
		{"empty origins", "", 0, false},
		{"single origin", "http://example.com", 1, false},
		{"multiple origins", "http://example.com,http://test.com", 2, false},
		{"origins with spaces", "http://example.com , http://test.com", 2, false},
		{"mixed origins", "http://example.com,*,http://test.com", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = db.NewConfig(
				"", ":8080", "http://localhost:11434/v1",
				"http://localhost:8081", "http://localhost:8080/callback",
				"client-id", "client-secret", "test.db",
				tt.origins,
			)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   *db.Config
		hasError bool
	}{
		{"valid config", &db.Config{
			HTTPAddr:         ":8080",
			LLMProxyURL:      "http://localhost:11434/v1",
			OAuthURL:         "http://localhost:8081",
			OAuthCallbackURL: "http://localhost:8080/callback",
			OAuthClientID:    "client-id",
			OAuthClientSecret: "client-secret",
			DBPath:           "test.db",
		}, false},
		{"missing LLM proxy URL", &db.Config{
			LLMProxyURL:      "",
			OAuthURL:         "http://localhost:8081",
			OAuthCallbackURL: "http://localhost:8080/callback",
			OAuthClientID:    "client-id",
			OAuthClientSecret: "client-secret",
			DBPath:           "test.db",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.HTTPAddr == "" {
				tt.config.HTTPAddr = ":8080"
			}
			if tt.config.DBPath == "" {
				tt.config.DBPath = "test.db"
			}

			err := tt.config.Validate()
			if tt.hasError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestConfigCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db.NewConfig(
		"", tmpDir+":8080", "http://localhost:11434/v1",
		"http://localhost:8081", "http://localhost:8080/callback",
		"client-id", "client-secret", filepath.Join(tmpDir, "test.db"),
		"",
	)
}
