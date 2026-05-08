# OAuth2 Proxy for Local LLM

Прокси для подключения к локальной LLM с OAuth2 аутентификацией.

## Установка зависимостей

```bash
go mod download
```

## Запуск

```bash
go run main.go
```

## Использование

1. Подключитесь через прокси на порту 8080:

```bash
curl http://localhost:8080/api/generate -H "Authorization: Bearer YOUR_TOKEN"
```

2.或直接使用:

```bash
go run main.go
```

## Конфигурация

- **LLM URL**: `http://localhost:11434` (по умолчанию)
- **OAuth URL**: `http://localhost:9000/oauth` (по умолчанию)
- **Прокси порт**: `:8080`

## OAuth2 Flow

Прокси использует client credentials flow для получения токенов.

## Безопасность

Для локального развития отключена проверка TLS (`InsecureSkipVerify: true`).
В продакшене необходимо включить проверку сертификатов.
