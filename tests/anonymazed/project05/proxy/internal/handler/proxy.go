package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/middleware"
)

type ProxyHandler struct {
	cfg          *config.LLMConfig
	oauthHandler *OAuthHandler
}

func NewProxyHandler(cfg *config.LLMConfig, oauthHandler *OAuthHandler) *ProxyHandler {
	return &ProxyHandler{
		cfg:          cfg,
		oauthHandler: oauthHandler,
	}
}

// LLMHandler обрабатывает запросы к LLM API
func (h *ProxyHandler) LLMHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем авторизацию
	ctx := r.Context()
	claims, ok := middleware.GetUserClaims(ctx)
	if !ok {
		http.Error(w, "Unauthorized: missing valid token", http.StatusUnauthorized)
		return
	}

	log.Printf("Proxying request for user: %s, path: %s", claims.UserID, r.URL.Path)

	// Создаем новый запрос для прокси
	proxyURL, err := h.buildProxyURL(r.URL)
	if err != nil {
		http.Error(w, "Invalid proxy URL", http.StatusInternalServerError)
		return
	}

	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, proxyURL.String(), r.Body)
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
	proxyReq.Header.Set("Authorization", "Bearer "+claims.AccessToken)

	// Создаем прокси
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.ModifyResponse = h.modifyResponse

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
	io.Copy(w, resp)
}

// buildProxyURL строит URL для прокси
func (h *ProxyHandler) buildProxyURL(rURL *url.URL) (*url.URL, error) {
	// Преобразуем OpenAI-compatible эндпоинт в стандартный формат
	proxyURL := url.URL{
		Scheme: h.cfg.APIURL,
		Host:   "localhost:11434",
	}

	// Преобразуем /v1/chat/completions в /api/chat/completions
	if strings.HasPrefix(rURL.Path, "/v1") {
		rURL.Path = strings.TrimPrefix(rURL.Path, "/v1")
		rURL.Path = "/api" + rURL.Path
	}

	return proxyURL.ResolveReference(rURL), nil
}

// modifyResponse модифицирует ответ прокси
func (h *ProxyHandler) modifyResponse(resp *http.Response) error {
	// Можно добавить логику для модификации ответа
	// Например, добавление заголовков или изменение тела ответа
	return nil
}

// HealthCheck endpoint для проверки здоровья сервиса
func (h *ProxyHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
