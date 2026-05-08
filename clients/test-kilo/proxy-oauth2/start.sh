#!/bin/bash

export PATH="/usr/local/opt/go/bin:$PATH"

# Start LLM server on port 11434 (ollama default)
echo "Starting Ollama LLM server..."
ollama serve &

# Wait for LLM to be ready
sleep 2

# Start OAuth2 proxy
./oauth2-proxy &

echo "OAuth2 Proxy started on http://localhost:8080"
echo "LLM endpoint: http://localhost:11434"
echo ""
echo "To connect through the proxy:"
echo "  curl -H 'Authorization: Bearer YOUR_TOKEN' http://localhost:8080/v1/chat/completions"
echo ""
echo "Press Ctrl+C to stop"

# Wait for user to press Ctrl+C
wait
