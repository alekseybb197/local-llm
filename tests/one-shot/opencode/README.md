# OAuth2 Proxy for LLM

Production-grade OAuth2 proxy for local LLM with OpenAI-compatible endpoint.

## Features

- OAuth2 Authorization Code Flow with PKCE
- GitHub OAuth integration
- Session management with timeout
- In-memory storage (for development)
- OpenAI-compatible chat completions proxy
- Health check endpoint
- Graceful shutdown

## Building

```bash
go build -o oauth2proxy .
```

## Running

```bash
# Default (connects to Ollama)
./oauth2proxy

# Custom LLM URL
LLM_URL=http://localhost:1234/v1 ./oauth2proxy
```

## Environment Variables

- `LLM_URL` - URL of the LLM API (default: http://localhost:11434/v1)

## API Endpoints

### Public

- `GET /login` - OAuth2 login redirect
- `GET /callback` - OAuth2 callback
- `GET /logout` - Logout
- `GET /` - Dashboard
- `GET /health` - Health check

### Protected (requires OAuth2 authentication)

- `POST /api/v1/chat/completions` - Chat completions
- `GET /api/v1/models` - List models

## Usage

1. Start the proxy:
   ```bash
   ./oauth2proxy
   ```

2. Visit http://localhost:8080/login

3. Authenticate with GitHub

4. Use the proxy for LLM requests

## Testing

```bash
go test ./...
```
