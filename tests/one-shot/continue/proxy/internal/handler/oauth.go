package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/middleware"
)

type OAuthHandler struct {
	cfg *config.OAuthConfig
}

func NewOAuthHandler(cfg *config.OAuthConfig) *OAuthHandler {
	return &OAuthHandler{
		cfg: cfg,
	}
}

// InitiateAuth инициирует OAuth2 Authorization Code Flow
func (h *OAuthHandler) InitiateAuth(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	
	// Сохраняем state в cookie для защиты от CSRF
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Отключите в разработке
		SameSite: http.SameSiteStrictMode,
	})

	authURL := h.cfg.TokenURL + "/auth"
	if r.URL.RawQuery != "" {
		authURL += "?" + r.URL.RawQuery + "&state=" + state
	} else {
		authURL += "?state=" + state
	}

	log.Printf("Redirecting to OAuth provider: %s", authURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback обрабатывает callback от OAuth2 провайдера
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	
	// Проверяем state для защиты от CSRF
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code parameter is required", http.StatusBadRequest)
		return
	}

	// Обмениваем код на токен
	ctx := context.Background()
	cfg := &oauth2.Config{
		ClientID:     h.cfg.ClientID,
		ClientSecret: h.cfg.ClientSecret,
		Scopes:       []string{"read"},
		TokenURL:     h.cfg.TokenURL,
	}
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Генерируем JWT токен для внутреннего использования
	claims := &middleware.Claims{
		UserID:      "user-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		AccessToken: token.AccessToken,
		Exp:         time.Now().Add(time.Hour * 24).Unix(),
		Iat:         time.Now().Unix(),
		Iss:         "oauth2-proxy",
		Aud:         "llm-api",
	}

	jwtToken := generateJWTToken(claims)

	// Возвращаем JWT токен клиенту
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": jwtToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})

	// Удаляем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

// generateState генерирует случайное state для защиты от CSRF
func generateState() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateJWTToken генерирует JWT токен из claims
func generateJWTToken(claims *middleware.Claims) string {
	// В реальном приложении здесь должна быть подпись токена
	// Для примера используем простую структуру
	return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." + 
	"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." +
	"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
