#!/bin/bash

. .env

echo -----
env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
echo -----

curl -s http://${LOCAL_LLM_HOST}:${LOCAL_LLM_PORT}/v1/chat/completions \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$LOCAL_LLM_ALIAS\", \"messages\": [{\"role\": \"user\", \"content\": \"def fib(n):\"}], \"max_tokens\": 2048}" | \
  jq

