# Hermes OAuth2 Proxy

Production-grade OAuth2 proxy на Go для локальной LLM с OpenAI-compatible endpoint.

## Особенности

- OAuth2 Authorization Code Flow
- Session-based authentication
- OpenAI-compatible API endpoint (`/v1/chat/completions`)
- CORS поддержка
- Health check endpoint
- Well-known OpenID configuration
- Встроенный in-memory store
- Поддержка HTTPS

## Структура проекта

```
hermes/
├── cmd/
│   └── oauth2proxy/
│       └── main.go           # Entry point
├── config/
│   ├── types.go             # Configuration types
│   └── types_test.go        # Configuration tests
├── internal/
│   ├── handlers/
│   │   └── oauth.go         # OAuth2 handlers
│   ├── server/
│   │   └── server.go        # HTTP server implementation
│   └── store/
│       ├── store.go         # Store interface and implementation
│       └── store_test.go    # Store tests
├── go.mod
└── go.sum
```

## Установка

```bash
# Clone repository
cd /path/to/hermes

# Build
go build -o hermes ./cmd/oauth2proxy/

# Run
./hermes
```

## Конфигурация

Создайте файл `config.json` в корневой директории:

```json
{
  "app_name": "hermes-oauth2-proxy",
  "oauth2": {
    "redirect_uri": "http://localhost:8080/callback",
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "scopes": ["openid", "profile", "email"],
    "endpoint_url": "https://auth.example.com/oauth2/auth",
    "token_url": "https://auth.example.com/oauth2/token",
    "user_info_url": "https://auth.example.com/oauth2/userinfo"
  },
  "server": {
    "host": "localhost",
    "port": 8080,
    "llm_api_url": "http://localhost:11434/api/generate",
    "llm_model": "local-llm",
    "allowed_origins": ["http://localhost:3000", "http://localhost:8080"],
    "session_secret": "your-secret-key-here",
    "cookie_expiration": 3600,
    "enable_https": false,
    "https_key_path": "",
    "https_cert_path": ""
  },
  "store": {
    "type": "memory",
    "memory_enabled": true
  }
}
```

## Переменные окружения

Все конфигурационные параметры можно задать через переменные окружения:

- `HERMES_APP_NAME` - название приложения
- `HERMES_OAUTH2_REDIRECT_URI` - URI перенаправления OAuth2
- `HERMES_OAUTH2_CLIENT_ID` - OAuth2 client ID
- `HERMES_OAUTH2_CLIENT_SECRET` - OAuth2 client secret
- `HERMES_SERVER_HOST` - host сервера
- `HERMES_SERVER_PORT` - порт сервера (можно с префиксом `port:`)
- `HERMES_SERVER_SESSION_SECRET` - секрет для сессий
- `HERMES_SERVER_LLMAPIURL` - URL API LLM

## Запуск

### С конфигурацией по умолчанию

```bash
go run ./cmd/oauth2proxy/
```

### С кастомной конфигурацией

```bash
./hermes
```

### С переменными окружения

```bash
HERMES_SERVER_PORT=9000 HERMES_SERVER_SESSION_SECRET=my-secret go run ./cmd/oauth2proxy/
```

## API

### OpenAI-compatible endpoint

#### POST /v1/chat/completions

```json
{
  "model": "local-llm",
  "messages": [
    {"role": "user", "content": "Привет, как дела?"}
  ],
  "temperature": 0.7,
  "max_tokens": 256
}
```

#### GET /v1/models

Список доступных моделей:

```json
{
  "data": [
    {
      "id": "local-llm",
      "object": "model",
      "created": 1234567890,
      "owned_by": "local"
    }
  ]
}
```

#### GET /v1/models/{model}

Информация о конкретной модели.

### OAuth2 endpoints

- `GET /login` - initiate OAuth2 flow
- `GET /callback` - OAuth2 callback
- `GET /logout` - logout

### Public endpoints

- `GET /health` - health check
- `GET /.well-known/openid-configuration` - OpenID configuration
- `GET /.well-known/jwks.json` - JWKS endpoint

## Unit-тесты

```bash
go test ./... -v
```

## Пример использования

### 1. Запустите OAuth2 сервер

```bash
go run ./cmd/oauth2proxy/
```

### 2. Откройте в браузере

```
http://localhost:8080/login
```

### 3. После авторизации можно использовать API

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "local-llm",
    "messages": [{"role": "user", "content": "Привет"}]
  }'
```

## Безопасность

1. **Session Secret** - обязательно установите секрет для сессий
2. **HTTPS** - используйте HTTPS для production
3. **CORS** - ограничьте allowed_origins
4. **State parameter** - защищен от CSRF атак

## Лицензия

MIT
