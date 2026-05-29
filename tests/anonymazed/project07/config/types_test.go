package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AppName != "hermes-oauth2-proxy" {
		t.Errorf("Expected app name 'hermes-oauth2-proxy', got: %s", cfg.AppName)
	}

	if len(cfg.OAuth2.Scopes) == 0 {
		t.Error("Expected OAuth2 scopes to be set")
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got: %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got: %d", cfg.Server.Port)
	}

	if cfg.Server.CookieExpiration != 1*time.Hour {
		t.Errorf("Expected cookie expiration 1h, got: %v", cfg.Server.CookieExpiration)
	}

	if cfg.Store.Type != "memory" {
		t.Errorf("Expected store type 'memory', got: %s", cfg.Store.Type)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"

	config := DefaultConfig()
	config.AppName = "test-app"
	config.Server.Host = "test.example.com"
	config.Server.Port = 9000

	if err := writeTestConfig(configPath, config); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config from file
	loadedCfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if loadedCfg.AppName != "test-app" {
		t.Errorf("Expected app name 'test-app', got: %s", loadedCfg.AppName)
	}

	if loadedCfg.Server.Host != "test.example.com" {
		t.Errorf("Expected host 'test.example.com', got: %s", loadedCfg.Server.Host)
	}

	if loadedCfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got: %d", loadedCfg.Server.Port)
	}
}

func TestLoadFromFile_EnvOverrides(t *testing.T) {
	// Save original environment variables
	origAppName := os.Getenv("HERMES_APP_NAME")
	origHost := os.Getenv("HERMES_SERVER_HOST")
	origPort := os.Getenv("HERMES_SERVER_PORT")
	origSessionSecret := os.Getenv("HERMES_SERVER_SESSION_SECRET")

	// Set environment variables
	os.Setenv("HERMES_APP_NAME", "env-app-name")
	os.Setenv("HERMES_SERVER_HOST", "env-host.com")
	os.Setenv("HERMES_SERVER_PORT", "3000")
	os.Setenv("HERMES_SERVER_SESSION_SECRET", "test-secret")

	// Load config from empty string - should use env vars with defaults
	cfg, err := LoadFromFile("")
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Restore original environment variables
	if origAppName != "" {
		os.Setenv("HERMES_APP_NAME", origAppName)
	} else {
		os.Unsetenv("HERMES_APP_NAME")
	}
	if origHost != "" {
		os.Setenv("HERMES_SERVER_HOST", origHost)
	} else {
		os.Unsetenv("HERMES_SERVER_HOST")
	}
	if origPort != "" {
		os.Setenv("HERMES_SERVER_PORT", origPort)
	} else {
		os.Unsetenv("HERMES_SERVER_PORT")
	}
	if origSessionSecret != "" {
		os.Setenv("HERMES_SERVER_SESSION_SECRET", origSessionSecret)
	} else {
		os.Unsetenv("HERMES_SERVER_SESSION_SECRET")
	}

	// Check that environment variables were applied
	if cfg.AppName != "env-app-name" {
		t.Errorf("Expected app name 'env-app-name', got: %s", cfg.AppName)
	}

	if cfg.Server.Host != "env-host.com" {
		t.Errorf("Expected host 'env-host.com', got: %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got: %d", cfg.Server.Port)
	}

	if cfg.Server.SessionSecret != "test-secret" {
		t.Errorf("Expected session secret 'test-secret', got: %s", cfg.Server.SessionSecret)
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"simple port", "8080", 8080, false},
		{"with port prefix", "port:8080", 8080, false},
		{"with space", "port: 8080", 8080, false},
		{"invalid port", "abc", 0, true},
		{"negative port", "-1", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := parsePort(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if port != tt.expected {
					t.Errorf("Expected port %d, got: %d", tt.expected, port)
				}
			}
		})
	}
}

func writeTestConfig(path string, cfg *Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
