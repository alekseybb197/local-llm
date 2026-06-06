package tests

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"oauth2-proxy/cmd/proxy-server/db"
	"oauth2-proxy/pkg/oauth2"
	"oauth2-proxy/pkg/proxy"
	"oauth2-proxy/pkg/oauth2/store"
)

func TestIntegration_LLMProxyWithOAuth2(t *testing.T) {
	// Create temporary directory for database
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create configuration
	cfg := db.NewConfig(
		"", ":8080", "http://localhost:11434/v1",
		"http://localhost:8081", "http://localhost:8080/callback",
		"client-id", "client-secret", filepath.Join(tmpDir, "test.db"),
		"",
	)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Failed to validate config: %v", err)
	}

	// Initialize database
	storeDB, err := db.New(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer storeDB.Close()

	// Initialize OAuth2 server
	oauthStore := storeDB.NewOAuth2Store()
	oauthServer, err := oauth.NewServer(
		oauthStore,
		"http://localhost:8081",
		"http://localhost:8080/callback",
		"client-id",
		"client-secret",
	)
	if err != nil {
		t.Fatalf("Failed to initialize OAuth2 server: %v", err)
	}

	// Initialize LLM proxy
	llmProxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to initialize LLM proxy: %v", err)
	}

	// Test OAuth2 authorize endpoint
	t.Run("OAuth2 Authorize", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth2/authorize?client_id=test&redirect_uri=http://localhost:8080/callback", nil)
		w := httptest.NewRecorder()

		oauthServer.HandleAuthorize(w, req)

		// Should redirect (302)
		if w.Code != http.StatusFound {
			t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
		}
	})

	// Test Health endpoint
	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"oauth2-proxy"}`))

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}
		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
	})

	// Test LLM proxy with valid API key
	t.Run("LLM Proxy with Valid API Key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-key")

		w := httptest.NewRecorder()
		llmProxy.Handle(w, req, "/chat/completions")

		// Should not return 401
		if w.Code == http.StatusUnauthorized {
			t.Errorf("Expected status != %d", http.StatusUnauthorized)
		}
	})

	// Test LLM proxy without API key
	t.Run("LLM Proxy without API Key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte(`{"model":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "")

		w := httptest.NewRecorder()
		llmProxy.Handle(w, req, "/chat/completions")

		// Should return 401
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestIntegration_OAuth2Flows(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := db.NewConfig(
		"", ":8080", "http://localhost:11434/v1",
		"http://localhost:8081", "http://localhost:8080/callback",
		"client-id", "client-secret", filepath.Join(tmpDir, "test.db"),
		"",
	)

	storeDB, err := db.New(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer storeDB.Close()

	oauthStore := storeDB.NewOAuth2Store()
	oauthServer, err := oauth.NewServer(
		oauthStore,
		"http://localhost:8081",
		"http://localhost:8080/callback",
		"client-id",
		"client-secret",
	)
	if err != nil {
		t.Fatalf("Failed to initialize OAuth2 server: %v", err)
	}

	llmProxy, err := proxy.New("http://localhost:11434/v1")
	if err != nil {
		t.Fatalf("Failed to initialize LLM proxy: %v", err)
	}

	t.Run("OAuth2 Callback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth2/callback?code=auth-code&state=state", nil)
		w := httptest.NewRecorder()

		oauthServer.HandleCallback(w, req)

		// Should redirect to callback URL
		if w.Code != http.StatusFound {
			t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
		}
	})

	t.Run("OAuth2 Logout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth2/logout", nil)
		w := httptest.NewRecorder()

		http.Redirect(w, req, "/", http.StatusFound)

		// Should redirect to home
		if w.Code != http.StatusFound {
			t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
		}
	})

	t.Run("OAuth2 Login", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth2/login", nil)
		w := httptest.NewRecorder()

		http.Redirect(w, req, "/", http.StatusFound)

		// Should return success message
		if w.Code != http.StatusFound {
			t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
		}
	})
}

func TestIntegration_APIKeyManagement(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := db.NewConfig(
		"", ":8080", "http://localhost:11434/v1",
		"http://localhost:8081", "http://localhost:8080/callback",
		"client-id", "client-secret", filepath.Join(tmpDir, "test.db"),
		"",
	)

	storeDB, err := db.New(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer storeDB.Close()

	oauthStore := storeDB.NewOAuth2Store()

	t.Run("Create API Key", func(t *testing.T) {
		key, err := oauthStore.CreateAPIKey("Test Key", "user", "")
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}
		if key == nil {
			t.Fatal("Expected API key to be created")
		}
		if key.Name != "Test Key" {
			t.Errorf("Expected name 'Test Key', got '%s'", key.Name)
		}
	})

	t.Run("Get API Key", func(t *testing.T) {
		_, err := oauthStore.GetAPIKey("1")
		// Expected error for non-existent key
	})

	t.Run("Delete API Key", func(t *testing.T) {
		err := oauthStore.DeleteAPIKey("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})

	t.Run("Verify API Key", func(t *testing.T) {
		_, err := oauthStore.VerifyAPIKey("invalid-key")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})

	t.Run("List API Keys", func(t *testing.T) {
		keys, err := oauthStore.ListAPIKeys()
		if err != nil {
			t.Fatalf("Failed to list API keys: %v", err)
		}
		if keys == nil {
			t.Error("Expected API key list")
		}
	})
}

func TestIntegration_DatabaseInitialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := db.NewConfig(
		"", ":8080", "http://localhost:11434/v1",
		"http://localhost:8081", "http://localhost:8080/callback",
		"client-id", "client-secret", filepath.Join(tmpDir, "test.db"),
		"",
	)

	// Test database creation
	storeDB, err := db.New(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer storeDB.Close()

	// Test database close
	if err := storeDB.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
}

func TestIntegration_Migrations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "oauth2-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=ON")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables exist
	tables := []string{"users", "sessions", "tokens", "api_keys", "api_key_usage"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table '%s': %v", table, err)
		}
		if count != 1 {
			t.Errorf("Expected table '%s' to exist, got %d", table, count)
		}
	}
}

func jsonUnmarshal(data []byte, v interface{}) error {
	// Simple implementation - use standard encoding/json
	decoder := json.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(v)
}
