package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"oauth2proxy/store"
	"oauth2proxy/models"
)

// generateStateToken creates a random state token for CSRF protection
func generateStateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge creates PKCE code challenge
func generateCodeChallenge(codeVerifier string) (string, error) {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

// LoginHandler handles the OAuth2 authorization URL generation
func LoginHandler(stateStore store.SessionStore, userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stateToken, err := generateStateToken()
		if err != nil {
			http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
			return
		}

		codeVerifier, err := generateCodeVerifier()
		if err != nil {
			http.Error(w, "Failed to generate code verifier", http.StatusInternalServerError)
			return
		}

		codeChallenge, err := generateCodeChallenge(codeVerifier)
		if err != nil {
			http.Error(w, "Failed to generate code challenge", http.StatusInternalServerError)
			return
		}

		session := &models.Session{
			StateToken: stateToken,
			UserID:     "",
			CreatedAt:  time.Now(),
		}

		if err := stateStore.StoreSession(session); err != nil {
			http.Error(w, "Failed to store session", http.StatusInternalServerError)
			return
		}

		authURL := fmt.Sprintf(
			"https://github.com/login/oauth/authorize?"+
				"client_id=%s&"+
				"redirect_uri=%s&"+
				"response_type=code&"+
				"scope=public_repo&"+
				"state=%s&"+
				"code_challenge=%s&"+
				"code_challenge_method=S256",
			models.ClientID,
			models.RedirectURI,
			stateToken,
			codeChallenge,
		)

		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// CallbackHandler handles the OAuth2 callback
func CallbackHandler(stateStore store.SessionStore, userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			http.Error(w, "Authorization code not provided", http.StatusBadRequest)
			return
		}

		// Validate state token
		session, err := stateStore.GetSession(state)
		if err != nil {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Validate state token hasn't expired (10 minutes)
		if time.Since(session.CreatedAt) > 10*time.Minute {
			http.Error(w, "State token expired", http.StatusForbidden)
			return
		}

		// Get user with code
		_, err = userStore.GetUserByCode(code, session.UserID)
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusBadRequest)
			return
		}

		// Clear session
		if err := stateStore.ClearSession(state); err != nil {
			log.Printf("Failed to clear session: %v", err)
		}

		// Redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

// LogoutHandler handles user logout
func LogoutHandler(sessionStore store.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear user session if provided
		userID := r.URL.Query().Get("user_id")
		if userID != "" {
			// In a real implementation, you'd clear the user session
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// DashboardHandler returns the main dashboard
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
	<title>OAuth2 Proxy</title>
</head>
<body>
	<h1>OAuth2 Proxy for LLM</h1>
	<p>Welcome to the proxy</p>
	<a href="/logout">Logout</a>
</body>
</html>
`))
}

// HealthHandler provides health check endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

type StateStore interface {
	StoreSession(session *models.Session) error
	GetSession(stateToken string) (*models.Session, error)
	ClearSession(stateToken string) error
}

type UserStore interface {
	GetUserByCode(code string, stateToken string) (*models.User, error)
}

type SessionStore interface {
	StoreSession(session *models.Session) error
	GetSession(stateToken string) (*models.Session, error)
	ClearSession(stateToken string) error
}
