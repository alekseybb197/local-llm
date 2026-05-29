#!/bin/bash

source ${HOME}/llm-server/.env

log_file="$HOME/Library/Logs/llm-server.log"
# Функция логирования с временной меткой
log_event() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [LAUNCHER] $*" >> "$log_file"
}

log_event "Запуск llm-server (PID: $$)"

${HOME}/llm-server/app/llama-server -m ${HOME}/llm-server/models/$LOCAL_LLM_MODEL \
  --port $LOCAL_LLM_PORT \
  --host $LOCAL_LLM_HOST \
  --api-key $LOCAL_LLM_APIKEY \
  --batch-size $LOCAL_LLM_BATCH \
  --ctx-size $LOCAL_LLM_CONTEXT \
  --n-gpu-layers $LOCAL_LLM_NGLAYERS \
  --flash-attn on \
  --kv-unified \
  --no-warmup \
  --threads $LOCAL_LLM_THREADS \
  --jinja \
  --cache-type-k $LOCAL_LLM_CACHEK \
  --cache-type-v $LOCAL_LLM_CACHEV \
  --spec-type $LOCAL_LLM_SPEC \
  --spec-draft-n-max $LOCAL_LLM_NMAX \
  --timeout $LOCAL_LLM_TIMEOUT \
  --parallel $LOCAL_LLM_SLOTS \
  --alias $LOCAL_LLM_ALIAS \
  --cache-ram $LOCAL_LLM_CACHERAM \
  --verbosity $LOCAL_LLM_VERBOSITY \
  --reasoning $LOCAL_LLM_REASONING \
  --log-file "$log_file" 2>&1

EXIT_CODE=$?

# Логирование завершения
if [ $EXIT_CODE -eq 0 ]; then
    log_event "Сервер завершил работу штатно (код: $EXIT_CODE)"
else
    log_event "⚠️ Сервер упал с ошибкой (код: $EXIT_CODE) — launchd попытается перезапустить"
fi

# Возвращаем код выхода для launchd
exit $EXIT_CODE