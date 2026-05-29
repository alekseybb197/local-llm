package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Сохраняем старые значения
	oldPort := os.Getenv("PORT")
	oldHost := os.Getenv("HOST")
	defer func() {
		os.Setenv("PORT", oldPort)
		os.Setenv("HOST", oldHost)
	}()

	// Тест с дефолтными значениями
	cfg := Load()
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "change-this-in-production", cfg.JWT.SecretKey)

	// Тест с пользовательскими значениями
	os.Setenv("PORT", "9090")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	cfg = Load()
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, "test-secret-key", cfg.JWT.SecretKey)

	// Проверка времени экспирации
	assert.GreaterOrEqual(t, cfg.JWT.Expiration, time.Hour)
	assert.GreaterOrEqual(t, cfg.JWT.RefreshExp, time.Hour*24*7)
}

func TestGetEnv(t *testing.T) {
	// Сохраняем старые значения
	oldPort := os.Getenv("PORT")
	defer func() {
		os.Setenv("PORT", oldPort)
	}()

	// Тест с установленным значением
	os.Setenv("PORT", "8080")
	assert.Equal(t, "8080", getEnv("PORT", "default"))

	// Тест с дефолтным значением
	os.Unsetenv("PORT")
	assert.Equal(t, "default", getEnv("PORT", "default"))
}
