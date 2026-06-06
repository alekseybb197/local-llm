package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save current environment
	oldClientID := os.Getenv("OAUTH2_CLIENT_ID")
	oldClientSecret := os.Getenv("OAUTH2_CLIENT_SECRET")
	defer func() {
		os.Setenv("OAUTH2_CLIENT_ID", oldClientID)
		os.Setenv("OAUTH2_CLIENT_SECRET", oldClientSecret)
	}()

	// Test with environment variables
	os.Setenv("OAUTH2_CLIENT_ID", "test-client-id")
	os.Setenv("OAUTH2_CLIENT_SECRET", "test-client-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.OAuth2Config.ClientID != "test-client-id" {
		t.Errorf("Expected ClientID 'test-client-id', got '%s'", cfg.OAuth2Config.ClientID)
	}

	if cfg.OAuth2Config.ClientSecret != "test-client-secret" {
		t.Errorf("Expected ClientSecret 'test-client-secret', got '%s'", cfg.OAuth2Config.ClientSecret)
	}
}

func TestDurationFromEnv(t *testing.T) {
	// Test with valid duration
	os.Setenv("TEST_DURATION", "30s")
	if d := durationFromEnv("TEST_DURATION", 10*time.Second); d != 30*time.Second {
		t.Errorf("Expected 30s, got %v", d)
	}

	// Test with invalid duration (should use default)
	os.Setenv("TEST_DURATION_INVALID", "invalid")
	if d := durationFromEnv("TEST_DURATION_INVALID", 10*time.Second); d != 10*time.Second {
		t.Errorf("Expected default 10s, got %v", d)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with set environment variable
	os.Setenv("TEST_VAR", "test-value")
	if val := getEnv("TEST_VAR", "default"); val != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", val)
	}

	// Test with unset environment variable (should return default)
	os.Unsetenv("TEST_VAR")
	if val := getEnv("TEST_VAR", "default"); val != "default" {
		t.Errorf("Expected 'default', got '%s'", val)
	}
}

func TestDefaultConfiguration(t *testing.T) {
	// Ensure no conflicting environment variables
	os.Unsetenv("OAUTH2_CLIENT_ID")
	os.Unsetenv("OAUTH2_CLIENT_SECRET")
	os.Unsetenv("SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ServerConfig.Host != "localhost" {
		t.Errorf("Expected default Host 'localhost', got '%s'", cfg.ServerConfig.Host)
	}

	if cfg.ServerConfig.Port != 8080 {
		t.Errorf("Expected default Port 8080, got %d", cfg.ServerConfig.Port)
	}

	if cfg.ServerConfig.ReadTimeout != 30*time.Second {
		t.Errorf("Expected default ReadTimeout 30s, got %v", cfg.ServerConfig.ReadTimeout)
	}
}
