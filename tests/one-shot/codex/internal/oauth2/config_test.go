package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/sessions"
)

func TestNewOAuth2Handler(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"read", "write"},
	}

	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	if handler == nil {
		t.Fatal("NewOAuth2Handler() returned nil")
	}
	if handler.config.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", handler.config.ClientID, "test-client-id")
	}
	if handler.config.RedirectURL != "http://localhost:8080/callback" {
		t.Errorf("RedirectURL = %q, want %q", handler.config.RedirectURL, "http://localhost:8080/callback")
	}
}

func TestOAuth2Handler_IsHTTPS(t *testing.T) {
	tests := []struct {
		certFile string
		certKey  string
		want     bool
	}{
		{"", "", false},
		{"cert.pem", "key.pem", true},
		{"cert.pem", "", false},
		{"", "key.pem", false},
	}

	for _, tt := range tests {
		t.Run(tt.certFile+"-"+tt.certKey, func(t *testing.T) {
			config := Config{
				CertificateFile: tt.certFile,
				CertificateKey:  tt.certKey,
			}
			store := sessions.NewCookieStore([]byte("test-secret"))
			handler := NewOAuth2Handler(config, store)

			if handler.IsHTTPS() != tt.want {
				t.Errorf("IsHTTPS() = %v, want %v", handler.IsHTTPS(), tt.want)
			}
		})
	}
}

func TestOAuth2Handler_SessionTimeout(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	if handler.SessionTimeout() != 15*time.Minute {
		t.Errorf("SessionTimeout() = %v, want %v", handler.SessionTimeout(), 15*time.Minute)
	}
}

func TestOAuth2Handler_AuthCodeURL(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	url := handler.AuthCodeURL("")
	if url == "" {
		t.Fatal("AuthCodeURL() returned empty string")
	}
}

func TestOAuth2Handler_Exchange(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	ctx := context.Background()
	_, err := handler.Exchange(ctx, "test-code")
	if err == nil {
		t.Error("Exchange() should return error for invalid code")
	}
}

func TestOAuth2Handler_ValidateCode(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	tests := []struct {
		code    string
		wantErr bool
	}{
		{"valid-code", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := handler.ValidateCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCode(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestOAuth2Handler_ValidateRedirectURI(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	tests := []struct {
		uri     string
		wantErr bool
	}{
		{"http://localhost:8080/callback", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			err := handler.ValidateRedirectURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRedirectURI(%q) error = %v, wantErr %v", tt.uri, err, tt.wantErr)
			}
		})
	}
}

func TestOAuth2Handler_ParseScope(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	tests := []struct {
		scopes  string
		wantLen int
	}{
		{"read,write", 2},
		{"read", 1},
		{"", 0},
		{"read,write,delete", 3},
	}

	for _, tt := range tests {
		t.Run(tt.scopes, func(t *testing.T) {
			got := handler.ParseScope(tt.scopes)
			if len(got) != tt.wantLen {
				t.Errorf("ParseScope(%q) = %d, want %d", tt.scopes, len(got), tt.wantLen)
			}
		})
	}
}

func TestOAuth2Handler_BuildURL(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	tests := []struct {
		base    string
		params  string
		wantSub string
	}{
		{"http://localhost", "foo=bar", "http://localhost?foo=bar"},
		{"http://localhost", "", "http://localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			got := handler.BuildURL(tt.base, map[string][]string{"foo": {"bar"}})
			if len(got) < 1 {
				t.Fatalf("BuildURL() returned empty string")
			}
		})
	}
}

func TestOAuth2Handler_TokenSource(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	ctx := context.Background()
	ts := handler.TokenSource(ctx, nil)
	if ts == nil {
		t.Fatal("TokenSource() returned nil")
	}
}

func TestOAuth2Handler_RefreshToken(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	ctx := context.Background()
	token := &Token{
		AccessToken: "test-token",
	}
	_, err := handler.RefreshToken(ctx, token)
	if err == nil {
		t.Error("RefreshToken() should return error")
	}
}

func TestOAuth2Handler_ServeHTTP(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
}

func TestOAuth2Handler_IsValid(t *testing.T) {
	config := Config{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewOAuth2Handler(config, store)

	// Handler should be created successfully
	if handler == nil {
		t.Fatal("NewOAuth2Handler() returned nil")
	}
}
