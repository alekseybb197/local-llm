package handlers

import (
	"net/http"
)

// Error is a custom error wrapper
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func (e *Error) Response() {
	http.Error(http.NewResponse(), e.Message, e.Code)
}

// Errorf creates a new error with a formatted message
func Errorf(code int, format string, args ...interface{}) *Error {
	return NewError(code, format)
}
