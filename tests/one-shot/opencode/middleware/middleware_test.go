package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oauth2proxy/models"
	"oauth2proxy/store"
)

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	mw := NewAuthMiddleware(stateStore, userStore)

	// Test with valid session
	session := &models.Session{
		StateToken: "valid-state",
		UserID:     "valid-user",
		CreatedAt:  time.Now(),
	}
	ss, ok := stateStore.(store.SessionStore)
	if !ok {
		t.Fatal("Failed to cast to SessionStore")
	}
	ss.StoreSession(session)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/?state=valid-state", nil)
	w := httptest.NewRecorder()

	mw.RequireAuth(handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_NoSession(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	mw := NewAuthMiddleware(stateStore, userStore)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mw.RequireAuth(handler).ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}

	if w.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login, got: %s", w.Header().Get("Location"))
	}
}

func TestAuthMiddleware_RequireAuth_ExpiredSession(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	mw := NewAuthMiddleware(stateStore, userStore)

	// Create an expired session
	session := &models.Session{
		StateToken: "expired-state",
		UserID:     "expired-user",
		CreatedAt:  time.Now().Add(-20 * time.Minute),
	}
	ss, ok := stateStore.(store.SessionStore)
	if !ok {
		t.Fatal("Failed to cast to SessionStore")
	}
	ss.StoreSession(session)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/?state=expired-state", nil)
	w := httptest.NewRecorder()

	mw.RequireAuth(handler).ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}
}

func TestAuthMiddleware_RequireAuth_LoginPath(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	mw := NewAuthMiddleware(stateStore, userStore)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	mw.RequireAuth(handler).ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}

	if w.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login, got: %s", w.Header().Get("Location"))
	}
}

func TestAuthMiddleware_RequireAuth_Unauthorized(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	mw := NewAuthMiddleware(stateStore, userStore)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	mw.RequireAuth(handler).ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}
}
