# Local LLM Backend for MacBook Pro M4 (24GB)

## Обзор проекта

Проект представляет собой инфраструктуру для запуска и использования локального LLM-бэкенда на MacBook Pro M4 с 24GB Unified Memory. Решение оптимизировано для работы с плотными (dense) моделями через аппаратный ускоритель Metal.

[Полный документ с инструкциями и логированием](./macbook-pro-m4-24.md)

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    MacBook Pro M4 (24GB RAM)                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              llama.cpp (llama-server)                 │  │
│  │  • Qwen3.5-9B-Q4_K_S.gguf (5.01 GB)                   │  │
│  │  • 262k контекст, 30 tokens/s генерация               │  │
│  │  • GPU ускорение через Metal                          │  │
│  └───────────────────────────────────────────────────────┘  │
│                              │                              │
│                              ▼                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │          Auto-restart via launchd                       │  │
│  │  • com.user.llm-server.plist                           │  │
│  │  • Перезапуск только при крахе (не timeout)           │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Основные компоненты

### 1. **LLM Backend** (`llama-server`)
- **Фреймворк**: llama.cpp (b8999)
- **Модель**: Qwen3.5-9B-Q4_K_S.gguf (~5 GB)
- **Хост**: `http://127.0.0.1:8000`
- **Особенности**:
  - Flash Attention для длинных контекстов
  - Все слои на GPU (`--n-gpu-layers 999`)
  - KV-кэш: 4.25 GB (262k токенов, q8_0)
  - 8 CPU-тредов, 1 слот (единый диалог)

### 2. **Service Management**
- **launchd plist**: `com.user.llm-server.plist`
  - Автозапуск при входе
  - Перезапуск только при аварийном завершении
  - Throttle: 30 сек между попытками
- **launch.sh**: Запускает сервер с логированием

### 4. **Configuration**
- **`.env`**: Переменные окружения для llama-server
- **Config файлы**: Настройки для клиентов и прокси

## Установка моделей

```bash
# Локальные модели (GGUF формат)
models/
├── Qwen3.5-9B-Q4_K_S.gguf  # Основная модель (5.01 GB)
├── sample.env-Qwen3.5-9B-Q4_K_S.gguf  # Пример
├── sample.env-Qwen3.6-27B-UD-IQ3_XXS.gguf  # Пример 27B
└── models.txt  # Список доступных моделей
```

## Скрипты

| Скрипт | Назначение |
|--------|------------|
| `launch.sh` | Запуск сервера с логированием |
| `server.sh` | Тестовый запуск |
| `benchmark.sh` | Бенчмарк скорости (tokens/s) |
| `show_models.sh` | Список моделей на бэкенде |
| `start.sh`, `stop.sh` | Управление сервисом |

## Производительность

**Измерено на Qwen3.5-9B-Q4_K_S:**
- Обработка промпта: 374 токена/с
- Генерация: 30 токенов/с
- Память: ~10.1 GB (GPU), ~1.05 GB (CPU)
- Контекст: 262k токенов

## Память по компонентам

```
GPU (Unified Memory):
  • Модель (веса):    5.01 GB
  • KV-кэш:          4.25 GB
  • Compute buffer:  0.81 GB
  • RS buffer:        0.05 GB
  ──────────────────────────────
  Итого:            ~10.13 GB

CPU:
  • Остаток модели:   0.53 GB
  • Compute buffer:   0.52 GB
  ──────────────────────────────
  Итого:            ~1.05 GB
```

## Переменные окружения (`.env`)

```bash
export LOCAL_LLM_MODEL="${LOCAL_LLM_MODEL:-Qwen3.5-9B-Q4_K_S.gguf}"
export LOCAL_LLM_PORT="${LOCAL_LLM_PORT:-8000}"
export LOCAL_LLM_HOST="${LOCAL_LLM_HOST:-127.0.0.1}"
export LOCAL_LLM_APIKEY="${LOCAL_LLM_APIKEY:-token_as_a_lot_of_symbols}"
export LOCAL_LLM_BATCH="${LOCAL_LLM_BATCH:-512}"
export LOCAL_LLM_CONTEXT="${LOCAL_LLM_CONTEXT:-262144}"  # 262k токенов
export LOCAL_LLM_NGLAYERS="${LOCAL_LLM_NGLAYERS:-999}"   # все слои на GPU
export LOCAL_LLM_THREADS="${LOCAL_LLM_THREADS:-8}"
export LOCAL_LLM_CACHE="${LOCAL_LLM_CACHE:-q8_0}"        # k/v квантование
export LOCAL_LLM_TIMEOUT="${LOCAL_LLM_TIMEOUT:-900}"      # 15 мин
export LOCAL_LLM_SLOTS="${LOCAL_LLM_SLOTS:-1}"           # один диалог
export LOCAL_LLM_CACHERAM="${LOCAL_LLM_CACHERAM:-0}"     # без RAM-кэша
export LOCAL_LLM_ALIAS="${LOCAL_LLM_ALIAS:-qwen35}"
export LOCAL_LLM_REASONING="${LOCAL_LLM_REASONING:-off}" # без reasoning
export GGML_METAL_EMBEDDING=1  # эмбеддинги на GPU
```

## Запуск

### Быстрый старт

```bash
# Загрузить .env из проекта
source .env

# Запустить сервер
./launch.sh
```

### Автозапуск

```bash
# Установить launchd
launchctl load com.user.llm-server.plist

# Проверить статус
launchctl list | grep llm-server
```

### Управление

```bash
# Остановить
killall llama-server

# Проверить логи
tail -f ~/Library/Logs/llm-server.log
```

## Тестирование клиентов

```bash
# Проверка доступности бэкенда
./show_models.sh

# Тест запроса
./test1.sh

# Бенчмарк
./benchmark.sh
```

## Клиенты (clients/)

```
clients/
├── .claude/           # Конфиг Claude
├── .config/           # Общие конфиги (kilo, opencode)
├── .continue/         # Конфиг Continue.dev
├── .kilo/             # Конфиг Kilo
├── .qwen/             # Конфиг Qwen
├── test-claude/       # Тестовое задание для Claude агента
│   └── main.go        # OAuth2 client credentials
├── test-continue/     # Тестовое задание для Continue.dev агента
│   └── proxy/
│       └── oauth2/
│           └── main.go # JWT + token exchange
├── test-kilo/         # Тестовое задание для Kilo агента
│   └── proxy-oauth2/
│       ├── proxy.go   # JWKS и валидация токенов
│       └── start.sh   # Запуск
├── test-opencode/     # Тестовое задание для Opencode агента
│   └── oauth2-proxy-llm/
│       └── main.go    # State-based OAuth2
└── test-qwen/         # Тестовое задание для Qwen агента
    ├── main.go        # LLM client
    ├── llm_client.go  # Client для LLM API
    └── oauth2_token.go# OAuth2 token exchange
```

## API Endpoints (LLM Server)

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/v1/models` | GET | Список моделей |
| `/v1/chat/completions` | POST | Чат с моделью |
| `/v1/embeddings` | POST | Эмбеддинги |
| `/v1/generate` | POST | Генерация (legacy) |

## Безопасность

- API key для аутентификации запросов к LLM серверу
- Агенты используют токены для аутентификации
- Локальный сервер (127.0.0.1) — недоступен извне

## Требования

- macOS 14+ (Sonoma) или новее
- Apple Silicon (M4/M3/M2)
- Минимум 24GB RAM
- Go 1.21+ (для тестовых клиентов)

## Лицензия

Apache 2.0
