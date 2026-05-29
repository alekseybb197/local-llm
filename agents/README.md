# Настройка агентов для работы с LLM.

Агенты для анализа выбирались по возможности из верхней части рейтинга `Openroute` . Оригинальный источник `https://openrouter.ai/apps/category/coding`.

Все агенты устанавливаются с минимальной изоляцией и запускаются из специально подготовленных дироекторий, так как предполагается, что агент работает над проектом, расположенным в текущей папке и доверяет всем файлам внутри её.

Готовые примеры настройки агентов размещаются в папке __agents__ .

Везде далее и в примерах настроек следует заменить:
- __username__ на имя актуально учетной записи
- __token_as_a_lot_of_symbols__ на реально используемый токен

## Aider

`https://github.com/aider-ai/aider`

### Установка

```bash
❯ curl -LsSf https://aider.chat/install.sh | sh
```

### Настройка

`https://aider.chat/docs/llms/openai-compat.html`

```bash
export OPENAI_API_KEY=token_as_a_lot_of_simbols
export OPENAI_BASE_URL=http://127.0.0.1:8000/v1
```

### Проверка

```bash
❯ mkdir -p tmp/test-aider
❯ cd tmp/test-aider
❯ aider --model openai/qwen35
```

## Claude Code

`https://github.com/anthropics/claude-code`

### Установка

Устанавливаем `https://code.claude.com/docs/en/quickstart`

```bash
❯ brew install --cask claude-code
```

### Настройка

Файл с настройками в `~/.claude/settings.json`:

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8000",
    "ANTHROPIC_API_KEY": "token_as_a_lot_of_symbols",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}
```

### Проверка

```bash
❯ mkdir -p tmp/test-claude
❯ cd tmp/test-claude
```

---

Запуск без VPN не работает:

```bash
❯ claude --model qwen35
Welcome to Claude Code v2.1.118
…………………………………………………………………………………………………………………………………………………………

     *                                       █████▓▓░
                                 *         ███▓░     ░░
            ░░░░░░                        ███▓░
    ░░░   ░░░░░░░░░░                      ███▓░
   ░░░░░░░░░░░░░░░░░░░    *                ██▓░░      ▓
                                             ░▓▓███▓▓░
 *                                 ░░░░
                                 ░░░░░░░░
                               ░░░░░░░░░░░░░░░░
       █████████                                        *
      ██▄█████▄██                        *
       █████████      *
…………………█ █   █ █………………………………………………………………………………………………………………

 Unable to connect to Anthropic services

 Failed to connect to api.anthropic.com: ERR_BAD_REQUEST

 Please check your internet connection and network settings.

 Note: Claude Code might not be available in your country. Check supported countries at https://anthropic.com/supported-countries
 ```

 ---

После включения VPN , можно уже залогиниться и приступить к работе.

```bash
❯ source ~/.claude/.env
❯ claude --model qwen35
╭─── Claude Code v2.1.118 ───────────────────────────────────────────────────────────────────────────╮
│                                │ Tips for getting started                                          │
│          Welcome back!         │ Run /init to create a CLAUDE.md file with instructions for Claude │
│                                │ ───────────────────────────────────────────────────────────────── │
│             ▐▛███▜▌            │ Recent activity                                                   │
│            ▝▜█████▛▘           │ No recent activity                                                │
│              ▘▘ ▝▝             │                                                                   │
│                                │                                                                   │
│   qwen35 · API Usage Billing   │                                                                   │
│       ~/tmp/test-claude        │                                                                   │
╰────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

## Cline

https://github.com/cline/cline

Для VS Code есть одноименный плагин.

### Установка

```bash
❯ brew install cline
```

### Настройка

Происходит в диалоге при первом старте.

В завершении должен быть создан файл, определяющий бэкэнд LLM   `~/.cline/data/settings/providers.json` приблизительно такого содержания:

```json
{
  "version": 1,
  "lastUsedProvider": "openai-compatible",
  "providers": {
    "cline": {
      "settings": {
        "provider": "cline",
        "model": "qwen/qwen3.7-max"
      },
      "updatedAt": "2026-05-23T22:07:33.229Z",
      "tokenSource": "manual"
    },
    "openai-compatible": {
      "settings": {
        "provider": "openai-compatible",
        "apiKey": "token_as_a_lot_of_symbols",
        "model": "qwen35",
        "baseUrl": "http://127.0.0.1:8000/v1",
        "reasoning": {
          "enabled": false
        }
      },
      "updatedAt": "2026-05-23T22:08:35.476Z",
      "tokenSource": "manual"
    }
  }
}
```

### Проверка

```bash
❯ mkdir cline
❯ cd cline
❯ cline
```

Агент работает в TUI режиме.

## Codex

`https://github.com/openai/codex`

### Установка

```bash
❯ brew install --cask codex
```

### Настройка

Файл настроек аутентификации `~/.codex/auth.json`:

```json
{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "token_as_a_lot_of_symbols"
}
```

Файл настроек профилей `~/.codex/config.toml`:

```
[model_providers.litellm]
name = "LiteLLM → llama-server"
base_url = "http://127.0.0.1:8000/v1"
wire_api = "responses"
stream_idle_timeout_ms = 300000

[profiles.qwen35]
model_provider = "litellm"
model = "qwen35"
```

### Проверка

```bash
❯ mkdir -p tmp/test-codex
❯ cd tmp/test-codex
❯ codex --profile qwen35
```

## Continue.dev

Поставляется в формате расширения для VS Code `https://www.continue.dev/`

### Установка

В UI VS Code из библиотеки расширений.

### Настройка

Для continue.dev конфиг живёт в `~/.continue/config.json`. Далее файл настроек под установленный бэк:

```json
{
  "models": [
    {
      "title": "Qwen3.5-9B (local)",
      "provider": "openai",
      "model": "qwen35",
      "apiBase": "http://127.0.0.1:8000/v1",
      "apiKey": "token_as_a_lot_of_symbols",
      "contextLength": 262144,
      "completionOptions": {
        "temperature": 0.6,
        "topP": 0.95,
        "maxTokens": 8192,
        "presencePenalty": 0,
        "frequencyPenalty": 0
      },
      "requestOptions": {
        "timeout": 300000,
        "extraBodyProperties": {
          "enable_thinking": true,
          "thinking": {
            "budget_tokens": 4096
          }
        }
      }
    }
  ],

  "tabAutocompleteModel": {
    "title": "Qwen3.5-9B (autocomplete)",
    "provider": "openai",
    "model": "qwen35",
    "apiBase": "http://127.0.0.1:8000/v1",
    "apiKey": "token_as_a_lot_of_symbols",
    "contextLength": 65536,
    "completionOptions": {
      "temperature": 0.2,
      "maxTokens": 512
    },
    "requestOptions": {
      "timeout": 30000,
      "extraBodyProperties": {
        "enable_thinking": false
      }
    }
  },

  "embeddingsProvider": {
    "provider": "transformers.js"
  },

  "slashCommands": [
    { "name": "edit",    "description": "Edit selected code" },
    { "name": "comment", "description": "Add comments to code" },
    { "name": "share",   "description": "Export chat as markdown" },
    { "name": "cmd",     "description": "Generate a shell command" }
  ],

  "contextProviders": [
    { "name": "code" },
    { "name": "docs" },
    { "name": "diff" },
    { "name": "terminal" },
    { "name": "problems" },
    { "name": "folder" },
    { "name": "codebase" }
  ],

  "allowAnonymousTelemetry": false
}
```

---

**Что важно знать по каждому блоку:**

**`models`** — основная чат-модель. Thinking включён, параметры те же, что в qwen-code.

**`tabAutocompleteModel`** — отдельная конфигурация для инлайн-автодополнения. Здесь `enable_thinking: false` принципиально — thinking mode для автокомплита бесполезен (медленно, большой overhead), а `maxTokens: 512` отражает реальные нужды. Temperature 0.2 — детерминированнее для кода.

**`embeddingsProvider: transformers.js`** — работает локально в самом VS Code без дополнительного сервера, не нагружает llama-server. Используется для `@codebase` контекста.

**`contextProviders`** — включи `codebase` только если проект небольшой, он индексирует весь репо через embeddings. Для больших монорепо лучше убрать или настроить `.continueignore`.

---

**Важные моменты:**

1. **`tabAutocompleteModel`** — если `qwen35` не обучена на FIM (fill-in-the-middle) токены, автодополнение будет работать плохо. Для autocomplete обычно используют специализированные модели (Qwen2.5-Coder, DeepSeek-Coder, Codellama и т.д.).
2. **`embeddingsProvider: transformers.js`** — в Continue.dev этот провайдер работает только если расширение запущено в режиме с доступом к WASM/браузерному движку. Если индексация codebase не работает, замени на `"openai"` с тем же `apiBase` (если сервер поддерживает эмбеддинги) или используй отдельную модель эмбеддингов.


## DeepSeek-TUI

`https://github.com/Hmbown/DeepSeek-TUI`

### Установка

```bash
❯ brew tap Hmbown/deepseek-tui
❯ brew install deepseek-tui
❯ whereis DeepSeek-TUI
DeepSeek-TUI: /opt/homebrew/bin/DeepSeek-TUI
❯ whereis deepseek-tui
deepseek-tui: /opt/homebrew/bin/deepseek-tui
❯ whereis deepseek
deepseek: /opt/homebrew/bin/deepseek
```

### Настройка

Создаем запись провайдера в `~/.deepseek/config.toml`:

```bash
❯ deepseek auth set --provider openai --api-key "token_as_a_lot_of_simbols"
saved API key for openai to /Users/username/.deepseek/config.toml and file-based (~/.deepseek/secrets/)
```

Затем добавляем остальные ключи:

```
auth_mode = "api_key"
provider = "openai"

[http_headers]

[projects."/Users/username/tmp/oauth2/deepseek"]
trust_level = "trusted"

[providers.atlascloud.http_headers]

[providers.deepseek.http_headers]

[providers.fireworks.http_headers]

[providers.novita.http_headers]

[providers.nvidia_nim.http_headers]

[providers.ollama.http_headers]

[providers.openai]
api_key = "token_as_a_lot_of_simbols"
base_url = "http://127.0.0.1:8000/v1"
model = "qwen35"

[providers.openai.http_headers]

[providers.openrouter.http_headers]

[providers.sglang.http_headers]

[providers.vllm.http_headers]

[providers.wanjie_ark.http_headers]
```

### Проверка

```bash
❯ mkdir -p tmp/test-deepseek
❯ cd tmp/test-deepseek
❯ deepsek
```

Агент работает в TUI режиме.


## Hermes

`https://github.com/nousresearch/hermes-agent`

### Установка

```bash
❯ curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash
```

### Настройка

Настройка производится в диалоговом режиме в процессе установки:

```bash
Custom OpenAI-compatible endpoint configuration:

API base URL [e.g. https://api.example.com/v1]: http://127.0.0.1:8000/v1
API key [optional]: 
Verified endpoint via http://127.0.0.1:8000/v1/models (1 model(s) visible)
  Detected model: qwen35
  Use this model? [Y/n]: y
Context length in tokens [leave blank for auto-detect]: 262144
Display name [Local (127.0.0.1:8000)]: Local Qwen 3.5

Default model set to: qwen35 (via http://127.0.0.1:8000/v1)
  💾 Saved to custom providers as "Local Qwen 3.5" (edit in config.yaml)
```

В завершении установки.

```bash
┌─────────────────────────────────────────────────────────┐
│              ✓ Installation Complete!                   │
└─────────────────────────────────────────────────────────┘


📁 Your files:

   Config:    /Users/username/.hermes/config.yaml
   API Keys:  /Users/username/.hermes/.env
   Data:      /Users/username/.hermes/cron/, sessions/, logs/
   Code:      /Users/username/.hermes/hermes-agent

─────────────────────────────────────────────────────────

🚀 Commands:

   hermes              Start chatting
   hermes setup        Configure API keys & settings
   hermes config       View/edit configuration
   hermes config edit  Open config in editor
   hermes gateway install Install gateway service (messaging + cron)
   hermes update       Update to latest version

─────────────────────────────────────────────────────────

⚡ Reload your shell to use 'hermes' command:

   source ~/.zshrc
```

### Проверка

```bash
❯ mkdir -p tmp/test-hermes
❯ cd tmp/test-hermes
❯ source ~/.zshrc
❯ hermes
```

## Kilocode

`https://github.com/kilo-org/kilocode`

В VS Code также устанавливается расширение Kilo Code.

### Установка

Пакет установки `https://github.com/Kilo-Org/homebrew-tap`

```bash
❯ brew install Kilo-Org/homebrew-tap/kilo
```

### Настройка

1. С помощью файла конфигурации

Конфиг лежит в корне проекта или глобально, формат `opencode.json`. Для custom/local моделей обязательно указывать `limit.context` и `limit.output` — без них автоматическое управление контекстом отключается и разговор будет расти неограниченно до отказа провайдера.

`~/.kilo/opencode.json` (или в корне проекта `.kilo/opencode.json`):

```json
{
  "$schema": "https://app.kilo.ai/config.json",
  "model": "llama-local/qwen35",
  "provider": {
    "llama-local": {
      "name": "llama-server local",
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "token_as_a_lot_of_symbols"
      },
      "models": {
        "qwen35": {
          "id": "qwen35",
          "name": "Qwen3.5-9B (local)",
          "tool_call": true,
          "limit": {
            "context": 262144,
            "output": 8192
          }
        }
      }
    }
  }
}
```

---

**Важный момент про timeout:**

Для локальных моделей дефолтный таймаут 10 минут. Если модель медленная — увеличь: VS Code Extensions panel → Kilo Code gear → Settings → **API Request Timeout**.

После настройки проверь что llama-server запущен и модель доступна — в Kilo Code panel должен появиться провайдер `llama-server local` с моделью `Qwen3.5-9B (local)` в пикере.

2. Через UI

В VS Code открой Kilo Code panel → шестерёнка → **Providers** → листай вниз → **Custom provider**:

- **Provider ID:** `llama-local`
- **Name:** `llama-server local`
- **Base URL:** `http://127.0.0.1:8000/v1`
- **API Key:** `token_as_a_lot_of_symbols`

Затем в поле модели введи `qwen35`.

### Проверка

```bash
❯ mkdir -p tmp/test-kilo
❯ cd tmp/test-kilo
❯ kilo
```

Агент работает в TUI режиме.

## Kimi

`https://github.com/MoonshotAI/kimi-cli`

Есть плагин к Vscode __Kimi Code__.

### Установка

```bash
curl -LsSf https://code.kimi.com/install.sh | bash
```

Первый запуск может потребовать авторизации на сайте `https://www.kimi.com` 

### Настройка

Файл настроек `~/.kimi/config.toml`

```
default_model = "qwen35"
default_thinking = false
default_yolo = false
skip_afk_prompt_injection = false
default_plan_mode = false
default_editor = ""
theme = "dark"
show_thinking_stream = true
hooks = []
merge_all_available_skills = true
extra_skill_dirs = []
telemetry = true

[models.qwen35]
provider = "llamacpp"
model = "qwen35"
max_context_size = 262144

[providers.llamacpp]
type = "openai_legacy"
base_url = "http://127.0.0.1:8080/v1"
api_key = "token_as_a_lot_of_symbols"
```

Чтобы избежать конфликта с обязательным подключением к облачному сервису kimi надо в окружение добавить переменные:

```bash
❯ export OPENAI_BASE_URL="http://127.0.0.1:8000/v1"
❯ export OPENAI_API_KEY="token_as_a_lot_of_symbols"
```

Для настройки плагина Vscode эти же переменные следует вписать в параметры плагина как переменные окружения и дополнительно указать путь к исполняемому файлу kimi -- ~/.local/bin/kimi .

### Проверка

```bash
❯ mkdir -p tmp/test-kimi
❯ cd tmp/test-kimi
❯ kimi
```

## Opencode

`https://github.com/anomalyco/opencode`

### Установка

```bash
❯ brew install --cask opencode-desktop
```

Это десктопное приложение и установка cli происходит из меню после старта.

### Настройка

OpenCode использует тот же формат что и Kilo CLI (Kilo — форк OpenCode).

Конфиг `~/.config/opencode/opencode.json` (или в корне проекта `opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "llama-local/qwen35",
  "provider": {
    "llama-local": {
      "name": "llama-server local",
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "token_as_a_lot_of_symbols"
      },
      "models": {
        "qwen35": {
          "id": "qwen35",
          "name": "Qwen3.5-9B (local)",
          "tool_call": true,
          "limit": {
            "context": 262144,
            "output": 8192
          }
        }
      }
    }
  }
}
```

### Проверка

```bash
❯ mkdir -p tmp/test-opencode
❯ cd tmp/test-opencode
❯ opencode
```

Агент работает в TUI режиме.

## Pi

`https://github.com/earendil-works/pi`

### Установка

```bash
❯ curl -fsSL https://pi.dev/install.sh | sh
```

### Настройка

Локальная модель определяется в файле `~/.pi/agent/models.json`

```json
{
  "providers": {
    "ollama": {
      "baseUrl": "http://127.0.0.1:8000/v1",
      "api": "openai-completions",
      "apiKey": "token_as_a_lot_of_symbols",
      "models": [
        {
          "id": "qwen35",
          "name": "Qwen3.5 Local",
          "input": ["text"],
          "contextWindow": 262144,
          "maxTokens": 8192
        }
      ]
    }
  }
}
```

### Проверка

```bash
❯ mkdir test-pi
❯ cd test-pi
❯ pi
 pi v0.74.1
 escape interrupt · ctrl+c/ctrl+d clear/exit · / commands · ! bash · ctrl+o more
 Press ctrl+o to show full startup help and loaded resources.
                                
 Pi can explain its own features and look up its docs. Ask it how to use or extend Pi.

──────────────────────────────────────────────────────────────────────────────────
                                      
──────────────────────────────────────────────────────────────────────────────────
~/tmp/test-pi
0.0%/262k (auto)
```

## Openhands

`https://github.com/OpenHands/OpenHands`

### Установка

```bash
❯ curl -fsSL https://install.openhands.dev/install.sh | sh
```

### Настройка

Настройка происходит в TUI при старте или по комаде `/settings`. Для работы с локальным бэком настройку надо проводит в режиме `Advanced` и к указанию используемой модели добавлять интерфейс, в данном случае с `llama.cpp` следует указывать `openai`, и полностью `openai/qwen35`.

Проверить настройки можно в файле `~/.openhands/agent_settings.json`.

### Проверка

```bash
❯ mkdir openhands
❯ cd openhands
❯ openhands
```

Агент работает в TUI режиме.

## Qwen-code

`https://github.com/QwenLM/qwen-code`

Кроме cli агента есть еще расширение для Vscode, называется Qwen Code Companion. В работе использует настройки cli агента

### Установка

```bash
❯ brew install qwen-code
. . . .
==> Upgrading qwen-code
  0.15.2 -> 0.15.6 
==> Pouring qwen-code--0.15.6.arm64_sequoia.bottle.tar.gz
🍺  /opt/homebrew/Cellar/qwen-code/0.15.6: 114 files, 30.5MB
==> Running `brew cleanup qwen-code`...
Disable this behaviour by setting `HOMEBREW_NO_INSTALL_CLEANUP=1`.
Hide these hints with `HOMEBREW_NO_ENV_HINTS=1` (see `man brew`).
Removing: /opt/homebrew/Cellar/qwen-code/0.15.2... (111 files, 29.7MB)
Removing: /Users/username/Library/Caches/Homebrew/qwen-code_bottle_manifest--0.15.2... (36.7KB)
Removing: /Users/username/Library/Caches/Homebrew/qwen-code--0.15.2... (6.1MB)
```

### Настройка

Описание конфигурации агента расположено в `~/.qwen/settings.json`

```json
{
  "modelProviders": {
    "openai": [
      {
        "id": "qwen35",
        "name": "Qwen3.5-9B-Q4_K_S",
        "baseUrl": "http://127.0.0.1:8000",
        "description": "Local Qwen3.5-9B",
        "envKey": "LOCAL_LLM_APIKEY",
        "generationConfig": {
          "timeout": 300000,
          "maxRetries": 1,
          "contextWindowSize": "262144",
          "samplingParams": {
            "temperature": 0.6,
            "top_p": 0.9,
            "max_tokens": 8192,
            "presence_penalty": 0,
            "frequency_penalty": 0
          },
          "extra_body": {
            "enable_thinking": true
          }
        }
      }
    ]
  },
  "env": {
    "LOCAL_LLM_APIKEY": "token_as_a_lot_of_symbols"
  },
  "security": {
    "auth": {
      "selectedType": "openai"
    }
  },
  "model": {
    "name": "${LOCAL_LLM_ALIAS}"
  },
  "tools": {
    "approvalMode": "default"
  },
  "$version": 3
}
```

### Проверка

Запускаем:

```bash
❯ mkdir -p tmp/test-qwen
❯ cd tmp/test-qwen
❯ qwen

   ▄▄▄▄▄▄  ▄▄     ▄▄ ▄▄▄▄▄▄▄ ▄▄▄    ▄▄   ┌──────────────────────────────────────────────────────────┐
  ██╔═══██╗██║    ██║██╔════╝████╗  ██║  │ >_ Qwen Code (v0.15.6)                                   │
  ██║   ██║██║ █╗ ██║█████╗  ██╔██╗ ██║  │                                                          │
  ██║▄▄ ██║██║███╗██║██╔══╝  ██║╚██╗██║  │ API Key | qwen35 (/model to change)                      │
  ╚██████╔╝╚███╔███╔╝███████╗██║ ╚████║  │ ~/tmp/test-qwen                                          │
   ╚══▀▀═╝  ╚══╝╚══╝ ╚══════╝╚═╝  ╚═══╝  └──────────────────────────────────────────────────────────┘

  Tips: You can run any shell commands from Qwen Code using ! (e.g. !ls).

─────────────────────────────────────────────────────────────────────────────────────────────────────
>   Type your message or @path/to/file
```

