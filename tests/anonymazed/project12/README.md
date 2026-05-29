# OAuth2 Proxy for Local LLM

A production-grade OAuth2 proxy server for local LLMs with OpenAI-compatible endpoint. Implements Authorization Code Flow with PKCE support.

## Features

- **OAuth2 Authorization Code Flow with PKCE**: Secure authorization with state and code challenges
- **OpenAI-compatible Proxy**: Transparent proxy for OpenAI-compatible LLM APIs
- **SQLite Backend**: Simple, portable database for user and API key management
- **API Key Management**: Create, list, and delete API keys for LLM access
- **CORS Support**: Configurable CORS origins
- **Graceful Shutdown**: Proper signal handling and cleanup
- **Comprehensive Testing**: Unit and integration tests

## Quick Start

### Build

```bash
make build
# or
go build -o bin/proxy-server ./cmd/proxy-server
```

### Run

```bash
make run
# or
./bin/proxy-server
```

### Test

```bash
make test
# or
go test -v ./...
```

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | HTTP server address |
| `-llm-proxy` | `http://localhost:11434/v1` | LLM proxy URL (OpenAI-compatible) |
| `-oauth-url` | `http://localhost:8081` | OAuth2 authorization URL |
| `-oauth-callback` | `http://localhost:8080/callback` | OAuth2 callback URL |
| `-oauth-client-id` | `proxy-client` | OAuth2 client ID |
| `-oauth-client-secret` | `proxy-secret` | OAuth2 client secret |
| `-db` | `proxy.db` | SQLite database path |
| `-cors-origins` | `*` | Comma-separated CORS origins |
| `-config` | `` | Path to configuration file |

### Example

```bash
./bin/proxy-server \
  --addr :8080 \
  --llm-proxy http://localhost:11434/v1 \
  --oauth-url http://localhost:8081 \
  --oauth-callback http://localhost:8080/callback
```

## API Endpoints

### OAuth2 Endpoints

- `GET /oauth2/authorize` - Initiate OAuth2 authorization
- `GET /oauth2/callback` - OAuth2 callback handler
- `GET /oauth2/logout` - Logout and clear session
- `GET /oauth2/login` - Login and create session

### Admin Endpoints

- `GET /oauth2/admin/api-keys` - List all API keys
- `POST /oauth2/admin/api-keys` - Create new API key
- `GET /oauth2/admin/api-keys/{id}` - Get API key details
- `DELETE /oauth2/admin/api-keys/{id}` - Delete API key

### Proxy Endpoints

- `POST /v1/*` - Proxy requests to LLM server

### Health Check

- `GET /health` - Health check endpoint

## API Key Usage

Use API keys to access the LLM proxy:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'
```

## Creating API Keys

```bash
curl -X POST http://localhost:8080/oauth2/admin/api-keys \
  -H "Authorization: Bearer admin" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "role": "user",
    "scope": "read"
  }'
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Client App    │────▶│   OAuth2 Server │────▶│    LLM Server   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              │ Proxy
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OAuth2 Proxy Server                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  OAuth2 Handler (Authorization Code Flow with PKCE)      │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │  │
│  │  │  Authorize  │  │   Callback  │  │   Token Exchange │  │  │
│  │  └─────────────┘  └─────────────┘  └──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  LLM Proxy (OpenAI-compatible)                           │  │
│  │  ┌─────────────┐  ┌───────────────────────────────────┐  │  │
│  │  │  Auth Check │  │  Request/Response Forwarding      │  │  │
│  │  └─────────────┘  └───────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  API Key Management                                      │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │  │
│  │  │  Create Key │  │   List Keys │  │   Delete Key     │  │  │
│  │  └─────────────┘  └─────────────┘  └──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  SQLite Database                                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │  │
│  │  │  Users      │  │  API Keys   │  │  Tokens          │  │  │
│  │  └─────────────┘  └─────────────┘  └──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Database Schema

- `users` - OAuth2 user accounts
- `sessions` - OAuth2 sessions
- `tokens` - OAuth2 access and refresh tokens
- `api_keys` - API keys for LLM access
- `api_key_usage` - Usage tracking for API keys

## Security Considerations

- **PKCE**: Implements PKCE (Proof Key for Code Exchange) for added security
- **State Token**: Prevents CSRF attacks
- **API Key Hashing**: API keys are hashed before storage
- **Session Management**: Secure session handling with HttpOnly cookies
- **CORS**: Configurable CORS origins for cross-origin requests

## Testing

The project includes comprehensive unit and integration tests:

- `tests/config_test.go` - Configuration tests
- `tests/store_test.go` - Store and database tests
- `tests/proxy_test.go` - LLM proxy tests
- `tests/integration_test.go` - Integration tests

Run tests with:
```bash
go test -v ./...
```

## License

MIT License
