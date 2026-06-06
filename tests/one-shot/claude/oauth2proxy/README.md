# OAuth2 Proxy for Local LLM

Production-grade OAuth2 proxy server for securing access to local LLM endpoints (Ollama, Mistral, etc.).

## Features

- **Authorization Code Flow**: Implements full OAuth2.0 authorization code flow with PKCE support
- **Session Management**: Secure cookie-based sessions with configurable timeouts
- **CORS Support**: Configurable CORS headers for browser clients
- **Health Checks**: `/health` endpoint for load balancers and monitoring
- **Skip Auth Mode**: Optional authentication bypass for local development
- **Production-Ready**: Built with security best practices, non-root containers, proper TLS support

## Quick Start

### 1. Install Dependencies

```bash
cd oauth2proxy
go mod download
```

### 2. Configure

Create `.oauth2config.json` in the project directory:

```json
{
  "listen_addr": "127.0.0.1:8080",
  "llm_api_addr": "http://127.0.0.1:11434",
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "auth_server_url": "http://localhost:3000",
  "scopes": ["openid", "profile", "email"],
  "allowed_origins": ["http://localhost:3000", "https://yourdomain.com"],
  "allowed_paths": ["/api/*"]
}
```

### 3. Run

```bash
# Debug mode
go run .

# Production build
go build -o oauth2proxy .
./oauth2proxy
```

### 4. Test

```bash
# Health check
curl http://localhost:8080/health

# API proxy (requires auth)
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/v1/models
```

### 5. Docker

```bash
docker build -t oauth2proxy .
docker run -p 8080:8080 --name oauth2proxy oauth2proxy
```

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `listen_addr` | string | `127.0.0.1:8080` | Address to bind the server |
| `llm_api_addr` | string | `http://127.0.0.1:11434` | Target LLM API endpoint |
| `client_id` | string | (required) | OAuth2 client ID from auth server |
| `client_secret` | string | (required) | OAuth2 client secret |
| `auth_server_url` | string | `http://localhost:3000` | OAuth2 authorization server URL |
| `scopes` | string[] | `["openid","profile","email"]` | Requested OAuth2 scopes |
| `skip_auth` | boolean | `false` | Bypass authentication (dev only) |
| `session_max_age` | int | `900` | Session timeout in seconds |
| `tls_enabled` | boolean | `false` | Enable TLS |
| `cert_file` | string | - | Path to TLS certificate |
| `key_file` | string | - | Path to TLS private key |
| `allowed_origins` | string[] | `[]` | CORS allowed origins |
| `allowed_paths` | string[] | `["/api/*"]` | Allowed API paths |

## OAuth2 Implementation

The proxy implements the Authorization Code Flow:

1. **Redirect to Auth**: User is redirected to OAuth2 server
2. **Authorization**: User grants permissions
3. **Code Exchange**: Proxy exchanges authorization code for tokens
4. **Token Storage**: Access token stored in secure cookies
5. **API Proxying**: Requests forwarded with Bearer token

## Security Features

- **PKCE Support**: PKCE code challenge for CSRF protection
- **HttpOnly Cookies**: Session cookies cannot be accessed by JavaScript
- **SameSite Cookies**: CSRF protection via SameSite attribute
- **Non-root Container**: Docker runs as non-root user
- **TLS Support**: Full TLS configuration for production
- **CORS Validation**: Origin header validation

## Health Check

```json
{
  "status": "ok",
  "timestamp": "2026-05-21T17:28:00Z"
}
```

## Development

```bash
# Run tests
go test ./...

# Run with race detector
go run -race .

# Build for production
go build -ldflags="-s -w" -o oauth2proxy .
```

## License

MIT
