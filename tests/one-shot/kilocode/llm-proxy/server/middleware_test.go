package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCORSHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CORS(CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	})(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("expected GET, POST, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	timeoutHandler := Timeout(100 * time.Millisecond)(handler)

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	timeoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("expected status 408, got %d", w.Code)
	}
}
