package handler

import (
	"encoding/json"
	"net/http"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/middleware"
)

// HTTPHandler управляет HTTP маршрутизацией
func HTTPHandler(cfg *config.Config, oauthHandler *OAuthHandler, proxyHandler *ProxyHandler) http.Handler {
	mux := http.NewServeMux()

	// OAuth2 endpoints
	mux.HandleFunc("/oauth/initiate", oauthHandler.InitiateAuth)
	mux.HandleFunc("/oauth/callback", oauthHandler.HandleCallback)

	// LLM API endpoints
	// Все /v1/* маршруты проксируются к LLM
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		http.HandlerFunc(proxyHandler.LLMHandler)(w, r)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"version": "1.0.0",
		})
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "OAuth2 Proxy for LLM API",
		"docs":    "http://localhost:8080/docs",
		"health":  "/health",
		"oauth":   "/oauth/initiate",
		})
	})

	// Применяем middleware авторизации к всем маршрутам, кроме /oauth/* и /health
	return middlewareWithExclusions(mux, cfg.JWT, []string{
		"/oauth/initiate",
		"/oauth/callback",
		"/health",
		"/",
	})
}

// middlewareWithExclusions применяет middleware с исключением определенных маршрутов
func middlewareWithExclusions(handler http.Handler, cfg config.JWTConfig, exclusions []string) http.Handler {
	authMW := middleware.AuthMiddleware(&cfg)

	// Создаем хендлер с исключением
	exclusionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, exclusion := range exclusions {
			if r.URL.Path == exclusion {
				handler.ServeHTTP(w, r)
				return
			}
		}
		// Применяем authMW
		authMW(next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		}))
	})

	return exclusionHandler
}
