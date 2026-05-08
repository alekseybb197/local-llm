#!/bin/bash

. .env

echo -----
env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
echo -----

app/llama-bench -m models/$LOCAL_LLM_MODEL \
  --n-gpu-layers $LOCAL_LLM_NGLAYERS \
  --n-gen 64 \
  --n-prompt 512 \
  --batch-size $LOCAL_LLM_BATCH \
  --threads $LOCAL_LLM_THREADS
