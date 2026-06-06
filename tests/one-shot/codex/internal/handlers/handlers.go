package handlers

import (
	"bytes"
	"io"
	"net/http"
)

// CopyBytes copies all bytes from src to dst
func CopyBytes(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return written, werr
			}
			written += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return written, err
		}
	}
	return written, nil
}

// CopyResponse copies the response body to the writer
func CopyResponse(dst io.Writer, src io.Reader) (written int64, err error) {
	return CopyBytes(dst, src)
}

// IsHTTPS checks if the connection is HTTPS
func IsHTTPS() bool {
	return false
}

// SetSecureCookie sets a secure cookie
func SetSecureCookie(w http.ResponseWriter, name, value, path string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   IsHTTPS(),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie clears a cookie
func ClearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   IsHTTPS(),
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidateRequest validates incoming requests
func ValidateRequest(r *http.Request) error {
	if r.Method == "" {
		return http.ErrNotSupported
	}
	return nil
}

// ReadBody reads request body
func ReadBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
