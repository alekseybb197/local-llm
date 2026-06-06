package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
}

func TestRouter_Add(t *testing.T) {
	router := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Add("/test", handler, []string{"GET"})
	if len(router.routes) != 1 {
		t.Fatalf("Add() expected 1 route, got %d", len(router.routes))
	}
}

func TestRouter_Match(t *testing.T) {
	router := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Add("/test", handler, []string{"GET"})

	tests := []struct {
		path   string
		method string
		want   bool
	}{
		{"/test", "GET", true},
		{"/test", "POST", false},
		{"/other", "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"-"+tt.method, func(t *testing.T) {
			h, found := router.Match(tt.path, tt.method)
			if found != tt.want {
				t.Errorf("Match(%q, %q) found = %v, want %v", tt.path, tt.method, found, tt.want)
			}
			if found && h == nil {
				t.Error("Match() returned nil handler")
			}
		})
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	router := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Add("/test", handler, []string{"GET"})

	tests := []struct {
		path   string
		method string
		wantCode int
	}{
		{"/test", "GET", http.StatusOK},
		{"/test", "POST", http.StatusMethodNotAllowed},
		{"/other", "GET", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path+"-"+tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("ServeHTTP() status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}
