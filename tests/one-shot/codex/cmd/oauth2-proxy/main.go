package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

const (
	AuthCookieName   = "oauth_state"
	UserCookieName   = "oauth_user"
	SessionTimeout   = 15 * time.Minute
	MaxRedirects     = 10
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	ProxyURL     string
	SessionSecret string
	TLSConfig    TLSConfig
	CustomCSS      string
	CustomHeader   string
}

type TLSConfig struct {
	CertificateFile string
	CertificateKey  string
}

func main() {
	config := parseEnv()
	if err := run(config); err != nil {
		log.Fatal(err)
	}
}

func parseEnv() Config {
	return Config{
		ClientID:     getEnv("CLIENT_ID", ""),
		ClientSecret: getEnv("CLIENT_SECRET", ""),
		RedirectURI:  getEnv("REDIRECT_URI", "http://localhost:8080/callback"),
		Scopes:       parseArray(getEnv("SCOPES", "pull,push"))(strings.Split(getEnv("SCOPES", "pull,push"), ",")),
		ProxyURL:     getEnv("PROXY_URL", "http://localhost:11434/v1"),
		SessionSecret: getEnv("SESSION_SECRET", "dev-session-secret-change-in-prod"),
		TLSConfig:    parseTLSConfig(),
		CustomCSS:      getEnv("CUSTOM_CSS", ""),
		CustomHeader:   getEnv("CUSTOM_HEADER", "X-OAuth2-Proxy"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseArray(s string) func([]string) []string {
	return func(arr []string) []string {
		result := make([]string, 0, len(arr))
		for _, s := range arr {
			s = strings.TrimSpace(s)
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	}
}

func parseTLSConfig() TLSConfig {
	return TLSConfig{
		CertificateFile: getEnv("CERT_FILE", ""),
		CertificateKey:  getEnv("CERT_KEY", ""),
	}
}

func run(cfg Config) error {
	if cfg.SessionSecret == "" {
		bytes := make([]byte, 32)
		rand.Read(bytes)
		cfg.SessionSecret = string(bytes[:32])
	}

	cookieStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       cfg.Scopes,
	}

	oauthHandler := &OAuth2Handler{
		config:         oauthConfig,
		store:          cookieStore,
		sessionTimeout: SessionTimeout,
		proxyURL:       cfg.ProxyURL,
		disableLocalState: false,
		disableTlsVerify:  false,
		customCSS:        cfg.CustomCSS,
		customHeader:     cfg.CustomHeader,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/oauth2/callback", oauthHandler.callback)
	mux.HandleFunc("/oauth2/logout", oauthHandler.logout)
	mux.HandleFunc("/proxy", oauthHandler.handleProxyRequest)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if user := oauthHandler.currentUser(r); user != nil {
			proxyURL := cfg.ProxyURL
			if r.URL.Query().Get("proxy") == "" {
				proxyURL = cfg.ProxyURL
			}
			http.Redirect(w, r, proxyURL, http.StatusSeeOther)
			return
		}

		html := oauthHandler.renderLoginPage()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if cfg.CustomCSS != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte("<style>" + cfg.CustomCSS + "</style>"))
		}
		w.Write([]byte(html))
	})

	if cfg.TLSConfig.CertificateFile != "" && cfg.TLSConfig.CertificateKey != "" {
		cert, _ := tls.LoadX509KeyPair(cfg.TLSConfig.CertificateFile, cfg.TLSConfig.CertificateKey)
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.NoClientCert,
			MinVersion:   tls.VersionTLS12,
		}

		server := &http.Server{
			Addr:         ":8080",
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
			TLSConfig:    tlsConfig,
		}

		ctx, cancel := context.WithCancel(context.Background())
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		go func() {
			select {
			case <-shutdown:
				log.Println("Shutting down server...")
			case <-ctx.Done():
				return
			}
			cancel()
		}()

		if err := server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			return fmt.Errorf("failed to start server: %w", err)
		}
	} else {
		server := &http.Server{
			Addr:         ":8080",
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		ctx, cancel := context.WithCancel(context.Background())
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		go func() {
			select {
			case <-shutdown:
				log.Println("Shutting down server...")
			case <-ctx.Done():
				return
			}
			cancel()
		}()

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			return fmt.Errorf("failed to start server: %w", err)
		}
	}

	log.Println("Server stopped")
	return nil
}

type OAuth2Handler struct {
	config         *oauth2.Config
	store          *sessions.CookieStore
	sessionTimeout time.Duration
	proxyURL       string
	disableLocalState bool
	disableTlsVerify  bool
	customCSS       string
	customHeader    string
}

func NewOAuth2Handler(cfg *oauth2.Config, store *sessions.CookieStore) *OAuth2Handler {
	return &OAuth2Handler{
		config:         cfg,
		store:          store,
		sessionTimeout: SessionTimeout,
		proxyURL:       "http://localhost:11434/v1",
		disableLocalState: false,
		disableTlsVerify:  false,
		customCSS:        "",
		customHeader:     "X-OAuth2-Proxy",
	}
}

func (h *OAuth2Handler) currentUser(r *http.Request) *User {
	session, _ := h.store.Get(r, "user")
	if session == nil {
		return nil
	}
	user, ok := session.Values["user"].(*User)
	if !ok || user == nil {
		return nil
	}
	return user
}

func (h *OAuth2Handler) saveUserToSession(w http.ResponseWriter, r *http.Request, user *User, token *oauth2.Token) error {
	session, _ := h.store.Get(r, "user")
	if session == nil {
		return fmt.Errorf("session not found")
	}
	session.Values["user"] = user
	session.Values["token"] = token
	session.Options = &sessions.Options{
		MaxAge:   int(SessionTimeout.Seconds()),
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	return session.Save(r, w)
}

func (h *OAuth2Handler) callback(w http.ResponseWriter, r *http.Request) {
	log.Println("OAuth2 callback received")

	if r.URL.Query().Get("error") != "" {
		log.Printf("OAuth2 error: %s", r.URL.Query().Get("error_description"))
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := h.config.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	user := &User{ID: "user-1", Name: "User", Email: "user@example.com", Verified: true}
	if err := h.saveUserToSession(w, r, user, token); err != nil {
		log.Printf("Failed to save user: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/proxy", http.StatusSeeOther)
}

func (h *OAuth2Handler) logout(w http.ResponseWriter, r *http.Request) {
	log.Println("Logout requested")
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     UserCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *OAuth2Handler) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("Proxy request received")
	proxyURL := r.URL.Query().Get("proxy")
	if proxyURL == "" {
		proxyURL = "http://localhost:11434/v1"
	}

	if h.currentUser(r) == nil {
		http.Redirect(w, r, "/oauth2/auth", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, proxyURL, http.StatusSeeOther)
}

func (h *OAuth2Handler) handleTokenRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("Token request received")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.currentUser(r) == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := http.Post(h.proxyURL, "application/json", r.Body)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (h *OAuth2Handler) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	log.Println("User info request received")

	if h.currentUser(r) == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user := h.currentUser(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"verified": user.Verified,
	})
}

func (h *OAuth2Handler) renderLoginPage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2 Proxy - LLM Access</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0;
            padding: 20px;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            max-width: 400px;
            width: 100%;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            text-align: center;
        }
        p {
            color: #666;
            text-align: center;
            margin-bottom: 30px;
        }
        .button {
            display: block;
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.4);
        }
        .button:active {
            transform: translateY(0);
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🤖 LLM Proxy</h1>
        <p>Sign in to access the local language model</p>
        <button class="button" onclick="window.location.href='/oauth2/auth'">Sign In</button>
    </div>
</body>
</html>
`
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}
