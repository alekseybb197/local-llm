# LLM OAuth2 Proxy

A Go-based OAuth2 proxy for connecting to a local LLM (Ollama-compatible) with authentication and rate limiting.

## Features

- OAuth2 authentication flow with state token validation
- Secure session management with encrypted cookies
- LLM request forwarding (Ollama-compatible API)
- Streaming response support
- Prometheus metrics
- Health check endpoint
- Multiple LLM API endpoints (generate, chat, embeddings)

## Prerequisites

- Go 1.21+
- A local LLM server (e.g., Ollama on port 11434)

## Configuration

Set the following environment variables:

```bash
# LLM endpoint
export LLM_HOST=localhost
export LLM_PORT=11434
export LLM_PATH=/api/generate

# OAuth2 settings
export OLLAMA_CLIENT_ID=llm-proxy-client
export OLLAMA_CLIENT_SECRET=your-secret-key
export OLLAMA_TOKEN_URL=http://localhost:11434/oauth/token

# Admin credentials (for session user)
export ADMIN_USER=admin
export ADMIN_PASS=password

# Server settings
export PORT=8080
export EXPOSE_METRICS=false
```

## Usage

### Start the proxy

```bash
go run *.go
```

### Connect with OAuth2

1. Visit: `http://localhost:8080/login?redirect=/api/generate`
2. Authenticate with the LLM's OAuth2 provider
3. Redirected back with an access token stored in a session cookie

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/login` | GET | OAuth2 login |
| `/oauth2/callback` | GET | OAuth2 callback |
| `/api/generate` | POST | LLM generation (authenticated) |
| `/api/generate/stream` | POST | Streaming generation |
| `/api/chat` | POST | Chat with LLM (authenticated) |
| `/api/embeddings` | POST | Embedding generation (authenticated) |
| `/metrics` | GET | Prometheus metrics |

### API Request Format

#### Generate

```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "prompt": "Hello, how are you?",
    "stream": false,
    "options": {
      "temperature": 0.7,
      "max_tokens": 256
    }
  }'
```

#### Streaming

```bash
curl -X POST http://localhost:8080/api/generate/stream \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "prompt": "Hello, how are you?",
    "stream": true
  }'
```

#### Chat

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false
  }'
```

#### Embeddings

```bash
curl -X POST http://localhost:8080/api/embeddings \
  -H "Authorization: Bearer <token>" \
  -d 'What is AI?'
```

## Architecture

```
┌─────────────┐     OAuth2 Flow      ┌─────────────┐
│   Client    │ ───────────────────> │   LLM       │
│             │                      │             │
└─────────────┘                      └─────────────┘
       ▲                                   │
       │                                   │
       │ OAuth2 Token                      │
       │                                   ▼
       │                          ┌─────────────┐
       │<─────── Protected API ───│   Proxy     │
       │                          │   (Go)      │
       └──────────────────────────└─────────────┘
```

## Security

- Secure cookie encryption with securecookie
- Session expiry validation (24 hours default)
- OAuth2 state token CSRF protection
- Bearer token authentication for LLM requests
- Rate limiting (TODO)

## Development

```bash
# Install dependencies
go mod download

# Run with debugging
go run -race *.go

# Build for production
go build -o llm-proxy *.go

# Run tests
go test -v ./...
```

## Docker

```bash
# Build image
docker build -t llm-proxy .

# Run
docker run -p 8080:8080 \
  -e LLM_HOST=localhost \
  -e LLM_PORT=11434 \
  llm-proxy
```

## Troubleshooting

### Session expired
Clear browser cookies and visit `/login` again.

### OAuth2 token exchange fails
Check that the LLM server has an OAuth2 token endpoint configured.

### Connection refused to LLM
Verify the LLM server is running and accessible on the configured port.

## License

MIT
