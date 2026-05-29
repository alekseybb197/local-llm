package tests

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"oauth2-proxy/pkg/oauth2/store"
)

func TestSQLiteStoreAPIKey(t *testing.T) {
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

	store := store.NewSQLiteStore(db)

	t.Run("CreateAPIKey", func(t *testing.T) {
		key, err := store.CreateAPIKey("Test Key", "user", "")
		if err != nil {
			t.Errorf("Failed to create API key: %v", err)
		}
		if key == nil {
			t.Error("Expected API key to be created")
		}
		if key.Name != "Test Key" {
			t.Errorf("Expected name 'Test Key', got '%s'", key.Name)
		}
		if key.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", key.Role)
		}
	})

	t.Run("ListAPIKeys", func(t *testing.T) {
		_, err := store.ListAPIKeys()
		if err != nil {
			t.Errorf("Failed to list API keys: %v", err)
		}
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		err := store.DeleteAPIKey("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})

	t.Run("VerifyAPIKey", func(t *testing.T) {
		_, err := store.VerifyAPIKey("invalid-key")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})
}

func TestSQLiteStoreUser(t *testing.T) {
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

	store := store.NewSQLiteStore(db)

	t.Run("GetOrCreateUser", func(t *testing.T) {
		user, err := store.GetOrCreateUser("testuser", "Test User", "test@example.com")
		if err != nil {
			t.Errorf("Failed to get or create user: %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
	})
}

func TestSQLiteStoreToken(t *testing.T) {
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

	store := store.NewSQLiteStore(db)

	t.Run("CreateToken", func(t *testing.T) {
		token, err := store.CreateToken("user1", "access-token-123", "refresh-token-456", "Bearer", "read", 3600)
		if err != nil {
			t.Errorf("Failed to create token: %v", err)
		}
		if token.AccessToken != "access-token-123" {
			t.Errorf("Expected access token 'access-token-123', got '%s'", token.AccessToken)
		}
	})

	t.Run("ValidateToken", func(t *testing.T) {
		_, err := store.ValidateToken("access-token-123", "read")
		if err != nil {
			t.Errorf("Failed to validate token: %v", err)
		}
	})

	t.Run("ValidateTokenInvalid", func(t *testing.T) {
		_, err := store.ValidateToken("invalid-token", "read")
		if err == nil {
			t.Error("Expected error for invalid token")
		}
	})

	t.Run("DeleteToken", func(t *testing.T) {
		err := store.DeleteToken("access-token-123")
		if err != nil {
			t.Errorf("Failed to delete token: %v", err)
		}
	})
}

func TestAPIKeyListMarshal(t *testing.T) {
	keys := store.APIKeyList{
		Keys: []*store.APIKey{
			{ID: "1", Name: "Key 1", Role: "user"},
			{ID: "2", Name: "Key 2", Role: "admin"},
		},
	}

	jsonData := keys.String()
	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON string")
	}

	if !contains(jsonData, `"keys"`) {
		t.Error("Expected JSON to contain 'keys' field")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
