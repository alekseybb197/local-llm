#!/bin/bash

. .env

echo -----
env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
echo -----

app/llama-server -m models/$LOCAL_LLM_MODEL \
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
  --reasoning $LOCAL_LLM_REASONING

