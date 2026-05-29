package main

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	os.Clearenv()
	
	config := DefaultConfig()
	
	if config.ServerPort != "8080" {
		t.Errorf("Default port should be 8080, got %s", config.ServerPort)
	}
	
	if config.SessionTimeout != 10*time.Minute {
		t.Errorf("Default session timeout should be 10m, got %v", config.SessionTimeout)
	}
	
	if len(config.Scopes) == 0 {
		t.Error("Default scopes should not be empty")
	}
}

func TestConfig_Validate(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != nil {
		t.Errorf("Valid config should not error: %v", err)
	}
}

func TestConfig_Validate_MissingClientID(t *testing.T) {
	config := &Config{
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != ErrMissingClientID {
		t.Errorf("Expected ErrMissingClientID, got %v", err)
	}
}

func TestConfig_Validate_MissingClientSecret(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != ErrMissingClientSecret {
		t.Errorf("Expected ErrMissingClientSecret, got %v", err)
	}
}

func TestConfig_Validate_MissingRedirectURI(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != ErrMissingRedirectURI {
		t.Errorf("Expected ErrMissingRedirectURI, got %v", err)
	}
}

func TestConfig_Validate_MissingAuthEndpoint(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		TokenEndpoint:    "http://auth/token",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != ErrMissingAuthEndpoint {
		t.Errorf("Expected ErrMissingAuthEndpoint, got %v", err)
	}
}

func TestConfig_Validate_MissingTokenEndpoint(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		StateSecret:    "test-secret",
	}
	
	if err := config.Validate(); err != ErrMissingTokenEndpoint {
		t.Errorf("Expected ErrMissingTokenEndpoint, got %v", err)
	}
}

func TestConfig_Validate_MissingStateSecret(t *testing.T) {
	config := &Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost/callback",
		AuthorizationEndpoint: "http://auth/oauth",
		TokenEndpoint:    "http://auth/token",
	}
	
	if err := config.Validate(); err != ErrMissingStateSecret {
		t.Errorf("Expected ErrMissingStateSecret, got %v", err)
	}
}
