package middleware

import (
	"net/http"
	"sync"
	"time"

	"oauth2proxy/store"
)

type AuthMiddleware struct {
	sessionStore store.SessionStore
	userStore    store.UserStore
	mu           sync.RWMutex
}

func NewAuthMiddleware(sessionStore store.SessionStore, userStore store.UserStore) *AuthMiddleware {
	return &AuthMiddleware{
		sessionStore: sessionStore,
		userStore:    userStore,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionToken := r.URL.Query().Get("state")

		if sessionToken == "" {
			// Check if it's a login request
			if r.URL.Path == "/login" {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			redirect := r.URL.RequestURI()
			if redirect == "/" {
				redirect = "/login"
			}
			http.Redirect(w, r, redirect, http.StatusFound)
			return
		}

		session, err := m.sessionStore.GetSession(sessionToken)
		if err != nil {
			redirect := r.URL.RequestURI()
			if redirect == "/" {
				redirect = "/login"
			}
			http.Redirect(w, r, redirect, http.StatusFound)
			return
		}

		// Check session expiry (10 minutes)
		if time.Since(session.CreatedAt) > 10*time.Minute {
			redirect := r.URL.RequestURI()
			if redirect == "/" {
				redirect = "/login"
			}
			http.Redirect(w, r, redirect, http.StatusFound)
			return
		}

		// Get user
		_, err = m.userStore.GetUserByCode("", session.UserID)
		if err != nil {
			redirect := r.URL.RequestURI()
			if redirect == "/" {
				redirect = "/login"
			}
			http.Redirect(w, r, redirect, http.StatusFound)
			return
		}

		// For now, just allow the request through
		// In a production system, you'd store the user in context
		next.ServeHTTP(w, r)
	})
}
