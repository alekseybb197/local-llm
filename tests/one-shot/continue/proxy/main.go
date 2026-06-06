package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/oauth2-proxy/internal/config"
	"github.com/example/oauth2-proxy/internal/handler"
	"github.com/example/oauth2-proxy/internal/middleware"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Создаем OAuth2 и proxy обработчики
	oauthHandler := handler.NewOAuthHandler(&cfg.OAuth)
	proxyHandler := handler.NewProxyHandler(&cfg.LLM, oauthHandler)

	// Создаем HTTP хендлер
	httpHandler := handler.HTTPHandler(cfg, oauthHandler, proxyHandler)

	// Создаем сервер
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер
	go func() {
		log.Printf("OAuth2 Proxy server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Обработка graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
