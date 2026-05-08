#!/bin/bash

. .env

#echo -----
#env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
#echo -----

curl -s -k http://$LOCAL_LLM_HOST:$LOCAL_LLM_PORT/v1/models \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" | jq

