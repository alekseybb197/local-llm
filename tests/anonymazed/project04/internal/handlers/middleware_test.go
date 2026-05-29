package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logger := Logger(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	logger.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Logger middleware status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	recovery := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	recovery.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Recovery middleware status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cors := CORS(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()

	cors.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("CORS middleware status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	rr2 := httptest.NewRecorder()
	cors.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("CORS middleware status = %d, want %d", rr2.Code, http.StatusOK)
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	auth := Auth(handler)

	tests := []struct {
		name     string
		authorize bool
		wantCode int
	}{
		{"no-auth", false, http.StatusUnauthorized},
		{"with-auth", true, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authorize {
				req.Header.Set("Authorization", "Bearer test-token")
			}
			rr := httptest.NewRecorder()

			auth.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("Auth middleware status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}
