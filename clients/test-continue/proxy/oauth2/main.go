package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// OAuth2 Credentials (получить у вашего провайдера)
var (
	ClientID     = os.Getenv("OAUTH_CLIENT_ID")
	ClientSecret = os.Getenv("OAUTH_CLIENT_SECRET")
	TokenURL     = os.Getenv("OAUTH_TOKEN_URL")
	LLMURL       = os.Getenv("LLM_API_URL")
	Port         = os.Getenv("PORT")
	if Port == "" {
		Port = "8080"
	}
)

// OAuth2 конфигурация
var oauthConfig = &oauth2.Config{
	ClientID:     ClientID,
	ClientSecret: ClientSecret,
	Scopes:       []string{"read"}, // Добавьте необходимые scopes
	TokenURL:     TokenURL,
}

func main() {
	// Создаем OAuth2 прокси
	mux := http.NewServeMux()

	// Кастомный хендлер для получения токена (для OAuth2 Authorization Code flow)
	mux.HandleFunc("/oauth/token", handleToken)

	// Прокси для LLM API
	mux.HandleFunc("/api/llm", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Парсим токен
		token, err := parseToken(authHeader)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Создаем новый запрос с токеном
		proxyURL, err := url.Parse(LLMURL + r.URL.Path)
		if err != nil {
			http.Error(w, "Invalid proxy URL", http.StatusInternalServerError)
			return
		}

		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Копируем заголовки (исключая Authorization)
		for k, vv := range r.Header {
			if k == "Authorization" {
				continue
			}
			for _, v := range vv {
				proxyReq.Header.Add(k, v)
			}
		}

		// Добавляем токен в заголовок
		proxyReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

		// Создаем прокси
		proxy := httputil.NewSingleHostReverseProxy(proxyURL)
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Можно модифицировать ответ здесь, если нужно
			return nil
		}

		// Выполняем прокси
		resp, err := proxy.ServeHTTP(w, proxyReq)
		if err != nil {
			http.Error(w, "Proxy error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки ответа
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		// Копируем тело ответа
		http.CopyResponse(w, resp, nil, nil, nil)
	})

	// Запускаем сервер
	addr := fmt.Sprintf(":%s", Port)
	log.Printf("OAuth2 Proxy server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// parseToken парсит JWT токен
func parseToken(authHeader string) (*oauth2.Token, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header")
	}

	tokenString := parts[1]
	// В реальном приложении здесь должна быть проверка ключа и секретного ключа
	// Для примера используем HMAC ключ
	key := []byte("your-secret-key") // Замените на реальный секретный ключ

	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Создаем OAuth2 токен на основе JWT
		return &oauth2.Token{
			AccessToken:  fmt.Sprintf("%v", claims["access_token"]),
			RefreshToken: fmt.Sprintf("%v", claims["refresh_token"]),
			Expiry:       time.Unix(int64(claims["exp"].(float64)), 0),
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// handleToken обрабатывает запрос на получение OAuth2 токена
func handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	// Обмениваем код на токен
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	// Возвращаем токен клиенту
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": token.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}
