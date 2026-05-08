#!/bin/bash

. .env

URL="http://${LOCAL_LLM_HOST}:${LOCAL_LLM_PORT}/v1/chat/completions"

echo "🔍 Тест 1: Короткий запрос (скорость ответа)"
time curl -s -X POST "$URL" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$LOCAL_LLM_ALIAS\",\"messages\":[{\"role\":\"user\",\"content\":\"Напиши функцию Python для быстрой сортировки\"}],\"max_tokens\":2048}" | \
  tee test2.json | \
  jq -r '.choices[0].message.content'

echo -e "\n🔍 Тест 2: Длинный контекст (загрузка 8K токенов)"
# Генерируем промпт без переносов в bash-переменной
PROMPT=$(python3 -c "print('import numpy as np\n' * 200)")
# Безопасная сборка JSON через jq
PAYLOAD=$(jq -n \
  --arg model "$LOCAL_LLM_ALIAS" \
  --arg content "$PROMPT # Оптимизируй этот код, убери дубликаты и добавь типизацию" \
  --argjson max_tokens 2048 \
  '{
    model: $model,
    messages: [{role: "user", content: $content}],
    max_tokens: $max_tokens
  }')
# Отправка
time curl -s -X POST "$URL" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" | \
  tee -a test2.json | \
  jq -r '.choices[0].message.content // "ОШИБКА: $(.error.message)"'

echo -e "\n💡 Проверьте Activity Monitor: Memory Pressure должен быть зеленым"
