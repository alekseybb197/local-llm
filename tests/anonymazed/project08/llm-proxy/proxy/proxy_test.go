package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProxyRequest(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test response"}`))
	}))
	defer ts.Close()

	// Create original request
	originalReq, _ := http.NewRequest("POST", ts.URL, nil)
	originalReq.Header.Set("Content-Type", "application/json")

	// Make proxy request
	resp, err := ProxyRequest(ts.URL, originalReq, "", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if len(resp.Body) == 0 {
		t.Error("expected response body")
	}
}

func TestProxyRequestWithHeaders(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("X-Test-Header")
		if origin != "test-value" {
			t.Errorf("expected header 'test-value', got '%s'", origin)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test response"}`))
	}))
	defer ts.Close()

	// Create original request
	originalReq, _ := http.NewRequest("POST", ts.URL, nil)
	originalReq.Header.Set("Content-Type", "application/json")

	// Make proxy request with headers
	resp, err := ProxyRequest(ts.URL, originalReq, "X-Test-Header:test-value", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
