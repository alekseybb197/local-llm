package store

import (
	"testing"
	"time"

	"oauth2proxy/models"
)

func TestSessionStore_StoreAndGet(t *testing.T) {
	store := NewSessionStore()
	session := &models.Session{
		StateToken: "test-state-token",
		UserID:     "test-user-id",
		CreatedAt:  time.Now(),
	}

	if err := store.StoreSession(session); err != nil {
		t.Fatalf("Failed to store session: %v", err)
	}

	retrieved, err := store.GetSession("test-state-token")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.StateToken != "test-state-token" {
		t.Errorf("Expected state token 'test-state-token', got '%s'", retrieved.StateToken)
	}
}

func TestSessionStore_GetNonExistent(t *testing.T) {
	store := NewSessionStore()
	_, err := store.GetSession("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent session, got nil")
	}
}

func TestSessionStore_Clear(t *testing.T) {
	store := NewSessionStore()
	session := &models.Session{
		StateToken: "test-state-token",
		UserID:     "test-user-id",
		CreatedAt:  time.Now(),
	}

	if err := store.StoreSession(session); err != nil {
		t.Fatalf("Failed to store session: %v", err)
	}

	if err := store.ClearSession("test-state-token"); err != nil {
		t.Fatalf("Failed to clear session: %v", err)
	}

	_, err := store.GetSession("test-state-token")
	if err == nil {
		t.Fatal("Expected error after clearing session, got nil")
	}
}

func TestUserStore_GetOrCreate(t *testing.T) {
	store := NewUserStore()
	user, err := store.GetUserByCode("test-code", "")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.ID != "test-code" {
		t.Errorf("Expected user ID 'test-code', got '%s'", user.ID)
	}

	// Get same user again
	user2, err := store.GetUserByCode("test-code", "")
	if err != nil {
		t.Fatalf("Failed to get user again: %v", err)
	}

	if user.ID != user2.ID {
		t.Errorf("Expected same user, got different IDs")
	}
}

func TestUserStore_GetNonExistentCode(t *testing.T) {
	store := NewUserStore()
	_, err := store.GetUserByCode("", "")
	if err == nil {
		t.Fatal("Expected error for non-existent code, got nil")
	}
}

func TestUserStore_ConcurrentAccess(t *testing.T) {
	store := NewUserStore()
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(code string) {
			_, err := store.GetUserByCode(code, "")
			if err != nil {
				t.Errorf("Failed to get user for code '%s': %v", code, err)
			}
			done <- true
		}(t.Name())
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	store := NewSessionStore()
	done := make(chan bool)

	for i := 0; i < 50; i++ {
		go func(token string) {
			if err := store.StoreSession(&models.Session{StateToken: token}); err != nil {
				t.Errorf("Failed to store session for token '%s': %v", token, err)
			}
			done <- true
		}(t.Name())
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}
