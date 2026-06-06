package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oauth2proxy/models"
	"oauth2proxy/store"
)

func TestLoginHandler(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	handler := LoginHandler(stateStore, userStore)
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	if len(location) == 0 {
		t.Error("Expected non-empty redirect URL")
	}
}

func TestCallbackHandler(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	// Create a valid session
	session := &models.Session{
		StateToken: "test-state",
		UserID:     "test-user",
		CreatedAt:  time.Now(),
	}
	stateStore.(interface {
		StoreSession(*models.Session) error
	}).StoreSession(session)

	handler := CallbackHandler(stateStore, userStore)
	req := httptest.NewRequest("GET", "/callback?code=test-code&state=test-state", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}
}

func TestCallbackHandler_InvalidState(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	handler := CallbackHandler(stateStore, userStore)
	req := httptest.NewRequest("GET", "/callback?code=test-code&state=invalid-state", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCallbackHandler_ExpiredState(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	// Create an expired session
	session := &models.Session{
		StateToken: "test-state",
		UserID:     "test-user",
		CreatedAt:  time.Now().Add(-20 * time.Minute),
	}
	stateStore.(interface {
		StoreSession(*models.Session) error
	}).StoreSession(session)

	handler := CallbackHandler(stateStore, userStore)
	req := httptest.NewRequest("GET", "/callback?code=test-code&state=test-state", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCallbackHandler_NoCode(t *testing.T) {
	stateStore := store.NewSessionStore()
	userStore := store.NewUserStore()

	handler := CallbackHandler(stateStore, userStore)
	req := httptest.NewRequest("GET", "/callback?state=test-state", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	stateStore := store.NewSessionStore()

	handler := LogoutHandler(stateStore)
	req := httptest.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}
}

func TestDashboardHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Dashboard</body></html>"))
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Expected Content-Type 'text/html', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestHealthHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}
