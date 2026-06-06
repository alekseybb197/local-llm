package server

import (
	"net/http"
	"strings"
	"time"
)

type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

func CORS(config CORSConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if len(config.AllowOrigins) > 0 {
				allowed := false
				for _, allowedOrigin := range config.AllowOrigins {
					if allowedOrigin == "*" || strings.Contains(origin, allowedOrigin) {
						allowed = true
						break
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			done := make(chan bool, 1)
			defer close(done)
			
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()
			
			select {
			case <-time.After(timeout):
				w.WriteHeader(http.StatusRequestTimeout)
				return
			case <-done:
			}
		})
	}
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
