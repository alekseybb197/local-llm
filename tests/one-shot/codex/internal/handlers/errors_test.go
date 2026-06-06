package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewError(t *testing.T) {
	err := NewError(http.StatusUnauthorized, "Unauthorized")
	if err == nil {
		t.Fatal("NewError() returned nil")
	}
	if err.Code != http.StatusUnauthorized {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusUnauthorized)
	}
	if err.Message != "Unauthorized" {
		t.Errorf("Message = %q, want %q", err.Message, "Unauthorized")
	}
}

func TestError_Error(t *testing.T) {
	err := NewError(http.StatusNotFound, "Not Found")
	got := err.Error()
	if got != "Not Found" {
		t.Errorf("Error() = %q, want %q", got, "Not Found")
	}
}

func TestErrorResponse(t *testing.T) {
	err := NewError(http.StatusBadRequest, "Bad Request")
	rr := httptest.NewRecorder()
	err.Response()

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Response() status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf(http.StatusInternalServerError, "Error: %s", "test")
	if err == nil {
		t.Fatal("Errorf() returned nil")
	}
	if err.Message != "Error: test" {
		t.Errorf("Message = %q, want %q", err.Message, "Error: test")
	}
}
