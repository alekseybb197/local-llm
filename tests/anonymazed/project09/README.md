# OAuth2 Proxy for Local LLM

Production-grade OAuth2 proxy server for local LLM with OpenAI-compatible endpoint. Implements Authorization Code Flow with support for any OAuth2 provider.

## Features

- **OAuth2 Authorization Code Flow** - Full OAuth2 compliance
- **JWT Token Validation** - Secure token handling with expiration
- **OpenAI-Compatible API** - Proxy to local LLM (ollama, llama.cpp, etc.)
- **Generic OAuth2 Provider Support** - Configurable for any OAuth2 provider
- **HTTPS Support** - TLS/SSL support for production
- **Production-Ready** - Logging, error handling, security
- **Refresh Token Support** - OAuth2 refresh_token grant type
- **Token Caching** - In-memory token storage with expiration

## Prerequisites

- Go 1.21+
- OAuth2 provider (GitHub, Google, etc.)

## Quick Start

1. **Install dependencies:**

```bash
go mod download
```

2. **Configure:**

```bash
cp config.json.example config.json
```

Edit `config.json` with your OAuth2 provider settings.

3. **Run the server:**

```bash
go run main.go
```

Server will start on `http://localhost:8080`

4. **Use the proxy:**

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     http://localhost:8080/v1/chat/completions \
     -d '{"model": "llama2", "messages": [{"role": "user", "content": "Hello"}]}'
```

## Endpoints

### Token Endpoint

```
POST /oauth/token/
Content-Type: application/json

{
  "grant_type": "authorization_code",
  "code": "AUTHORIZATION_CODE_FROM_PROVIDER",
  "redirect_uri": "http://localhost:8080/callback/"
}
```

Response:
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 7200,
  "refresh_token": "new-token-123",
  "scope": "read write"
}
```

### Refresh Token Endpoint

```
POST /oauth/token/
Content-Type: application/json

{
  "grant_type": "refresh_token",
  "refresh_token": "EXISTING_REFRESH_TOKEN",
  "client_id": "YOUR_CLIENT_ID",
  "client_secret": "YOUR_CLIENT_SECRET"
}
```

### User Info Endpoint

```
GET /oauth/userinfo/
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Response:
```json
{
  "id": "user-123",
  "email": "user@example.com",
  "name": "Test User",
  "username": "testuser"
}
```

### LLM Proxy Endpoint (OpenAI-Compatible)

```
POST /v1/chat/completions/
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "model": "llama2",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ]
}
```

### Authorization Page

```
GET /auth/?client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:8080/callback/
```

Redirects to OAuth2 provider for authentication.

### OAuth Callback

```
GET /callback/?code=AUTH_CODE&state=STATE
```

Handles OAuth2 callback and redirects to redirect_uri.

## Configuration

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "redirect_uri": "http://localhost:8080/callback/",
  "cert_file": "",
  "key_file": "",
  "client_id": "default-client-id",
  "client_secret": "default-client-secret",
  "scopes": ["read", "write"]
}
```

- `host` - Server host (default: 0.0.0.0)
- `port` - Server port (default: 8080)
- `redirect_uri` - OAuth2 redirect URI (must match provider settings)
- `cert_file` - TLS certificate file (optional, for HTTPS)
- `key_file` - TLS private key file (optional, for HTTPS)
- `client_id` - OAuth2 client ID (optional, uses default if not set)
- `client_secret` - OAuth2 client secret (optional, uses default if not set)
- `scopes` - OAuth2 scopes (default: ["read", "write"])

## OAuth2 Provider Setup

### GitHub OAuth2

1. Register your app at https://github.com/settings/developers
2. Set Authorization URL: `https://github.com/login/oauth/authorize`
3. Set Token URL: `https://github.com/login/oauth/access_token`
4. Set Redirect URI: `http://localhost:8080/callback/`

Example config:
```json
{
  "authorization_url": "https://github.com/login/oauth/authorize",
  "token_url": "https://github.com/login/oauth/access_token",
  "client_id": "YOUR_GITHUB_CLIENT_ID",
  "client_secret": "YOUR_GITHUB_CLIENT_SECRET"
}
```

### Google OAuth2

1. Create OAuth2 credentials at https://console.cloud.google.com/
2. Set Authorization URL: `https://accounts.google.com/o/oauth2/v2/auth`
3. Set Token URL: `https://oauth2.googleapis.com/token`
4. Set Redirect URI: `http://localhost:8080/callback/`

Example config:
```json
{
  "authorization_url": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_url": "https://oauth2.googleapis.com/token",
  "client_id": "YOUR_GOOGLE_CLIENT_ID",
  "client_secret": "YOUR_GOOGLE_CLIENT_SECRET"
}
```

### Generic OAuth2 Provider

Simply configure the provider URLs and credentials. The proxy supports any OAuth2 provider that implements the standard Authorization Code Flow.

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestValidateToken
```

## Security

- ✅ State parameter validation (CSRF protection)
- ✅ JWT token validation with expiration
- ✅ Refresh token support
- ✅ Token expiration checks
- ✅ HTTPS support (TLS configuration)
- ✅ Secure token storage
- ✅ Input validation and sanitization

## Development

```bash
# Build
go build -o oauth2-proxy .

# Run with verbose logging
go run main.go

# Run tests
go test -v ./...

# Check coverage
go test -cover ./...
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  OAuth2     │────▶│  OAuth2      │────▶│   LLM       │
│  Provider   │◀────│  Proxy       │◀────│  (ollama,   │
└─────────────┘     └──────────────┘     │  llama.cpp) │
    User Flow         JWT Handling         OpenAI API
```

## License

MIT
